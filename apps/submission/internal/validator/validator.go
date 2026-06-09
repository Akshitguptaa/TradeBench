package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// allowedMIME defines the file types we accept
var allowedMIME = map[string]bool{
	"application/octet-stream": true,
	"application/zip":          true,
	"text/x-csrc":              true,
	"text/x-c":                 true,
	"text/plain; charset=utf-8": true,
}

// allowedLanguages defines the programming languages we support
var allowedLanguages = map[string]bool{
	"cpp":    true,
	"rust":   true,
	"go":     true,
	"python": true,
}

type ValidationResult struct {
	Hash string
	MIME string
	Size int64
}

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size")
	ErrInvalidMIME     = errors.New("unsupported file type")
	ErrInvalidLanguage = errors.New("unsupported language; must be cpp, rust, go, or python")
)

// Validate checks if the uploaded file meets our security and size constraints
func Validate(content []byte, language string, maxSizeBytes int64) (*ValidationResult, error) {
	if !allowedLanguages[language] {
		return nil, ErrInvalidLanguage
	}

	size := int64(len(content))
	if size > maxSizeBytes {
		return nil, fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, size, maxSizeBytes)
	}

	mime := http.DetectContentType(content)
	if !allowedMIME[mime] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMIME, mime)
	}

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

func ValidateReader(r io.Reader, language string, maxSizeBytes int64) (*ValidationResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading upload body: %w", err)
	}
	return Validate(data, language, maxSizeBytes)
}
