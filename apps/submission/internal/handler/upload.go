// Package handler implements the HTTP upload handler for the submission service.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/tradebench/submission/internal/queue"
	"github.com/tradebench/submission/internal/status"
	"github.com/tradebench/submission/internal/store"
	"github.com/tradebench/submission/internal/validator"
)

// UploadHandler handles multipart binary uploads.
type UploadHandler struct {
	store        *store.MinIO
	producer     *queue.KafkaProducer
	tracker      *status.Tracker
	maxFileBytes int64
}

// NewUploadHandler creates a handler wired to the given store, Kafka producer, and status tracker.
func NewUploadHandler(s *store.MinIO, p *queue.KafkaProducer, t *status.Tracker, maxFileSizeMB int) *UploadHandler {
	return &UploadHandler{
		store:        s,
		producer:     p,
		tracker:      t,
		maxFileBytes: int64(maxFileSizeMB) * 1024 * 1024,
	}
}

// uploadResponse is the JSON body returned on a successful upload.
type uploadResponse struct {
	SubmissionID string `json:"submission_id"`
	Status       string `json:"status"`
	Hash         string `json:"sha256"`
}

// ServeHTTP processes POST multipart/form-data uploads.
//
// Expected form fields:
//   - file:          the binary to submit
//   - language:      "cpp" | "rust" | "go"
//   - contestant_id: the submitting contestant
func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Enforce max body size
	// limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileBytes+1024) // +1KB for form overhead

	if err := r.ParseMultipartForm(h.maxFileBytes); err != nil {
		http.Error(w, `{"error":"request too large or invalid multipart form"}`, http.StatusBadRequest)
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	// --- Extract fields ---
	// extract metadata from the multipart form
	language := r.FormValue("language")
	contestantID := r.FormValue("contestant_id")
	if language == "" || contestantID == "" {
		http.Error(w, `{"error":"language and contestant_id are required"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file field is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file into memory for validation
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"failed to read uploaded file"}`, http.StatusInternalServerError)
		return
	}

	// --- Validate ---
	// validate file content (size, mime type, language)
	result, err := validator.Validate(content, language, h.maxFileBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	log.Printf("upload: validated file=%s mime=%s size=%d hash=%s",
		header.Filename, result.MIME, result.Size, result.Hash)

	// --- Store in MinIO ---
	// generate a unique ID for the submission
	submissionID := uuid.New().String()
	ctx := context.Background()

	// upload the binary to MinIO
	s3Key, err := h.store.Upload(ctx, submissionID, header.Filename, bytes.NewReader(content), result.Size)
	if err != nil {
		log.Printf("upload: store error: %v", err)
		http.Error(w, `{"error":"failed to store submission"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("upload: stored submission_id=%s s3_key=%s", submissionID, s3Key)

	// --- Publish to Kafka ---
	event := queue.SubmissionEvent{
		SubmissionID: submissionID,
		ContestantID: contestantID,
		S3Key:        s3Key,
		Language:     language,
		SubmittedAt:  time.Now().UnixMilli(),
	}

	// publish an event to Kafka so the sandbox can pick it up
	if err := h.producer.Publish(ctx, event); err != nil {
		log.Printf("upload: kafka publish error: %v", err)
		// The file is stored; we can still return success and retry the event.
		// For the scaffold we log and continue.
	}

	log.Printf("upload: published submission.queued submission_id=%s", submissionID)

	// track the submission
	h.tracker.Set(submissionID, contestantID, status.Queued)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(uploadResponse{
		SubmissionID: submissionID,
		Status:       "queued",
		Hash:         result.Hash,
	})
}
