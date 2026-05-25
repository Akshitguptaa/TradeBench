// Package validator provides file validation for uploaded submission binaries.
package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// allowedMIME is the set of MIME types accepted for submission uploads.
var allowedMIME = map[string]bool{
	"application/octet-stream": true,
	"application/zip":          true,
	"text/x-csrc":              true,
	"text/x-c":                 true,
}

// allowedLanguages is the set of valid language identifiers.
var allowedLanguages = map[string]bool{
	"cpp":  true,
	"rust": true,
	"go":   true,
}

// ValidationResult holds the outcome of a successful validation pass.
type ValidationResult struct {
	Hash string // SHA-256 hex digest
	MIME string // detected MIME type
	Size int64  // file size in bytes
}

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size")
	ErrInvalidMIME     = errors.New("unsupported file type")
	ErrInvalidLanguage = errors.New("unsupported language; must be cpp, rust, or go")
)

// Validate checks the uploaded file contents against size, MIME, and language
// constraints. On success it returns a ValidationResult containing the SHA-256
// hash, detected MIME type, and file size.
//
// The provided reader must be seekable (or the caller must supply the full
// content). The function reads up to maxSizeBytes; anything larger is rejected.
func Validate(content []byte, language string, maxSizeBytes int64) (*ValidationResult, error) {
	// --- Language check ---
	if !allowedLanguages[language] {
		return nil, ErrInvalidLanguage
	}

	// --- Size check ---
	size := int64(len(content))
	if size > maxSizeBytes {
		return nil, fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, size, maxSizeBytes)
	}

	// --- MIME check ---
	mime := http.DetectContentType(content)
	if !allowedMIME[mime] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMIME, mime)
	}

	// --- SHA-256 hash ---
	h := sha256.New()
	if _, err := h.Write(content); err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	return &ValidationResult{
		Hash: hex.EncodeToString(h.Sum(nil)),
		MIME: mime,
		Size: size,
	}, nil
}

// ValidateReader is a convenience wrapper that reads the full body from r
// before delegating to Validate.
func ValidateReader(r io.Reader, language string, maxSizeBytes int64) (*ValidationResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading upload body: %w", err)
	}
	return Validate(data, language, maxSizeBytes)
}
