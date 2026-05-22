// Package store wraps the MinIO client for storing submission binaries.
package store

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO is a thin wrapper around the minio-go client for submission storage.
type MinIO struct {
	client *minio.Client
	bucket string
}

// NewMinIO creates a MinIO store client. It ensures the target bucket exists
// (creating it if necessary).
func NewMinIO(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIO, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio create bucket: %w", err)
		}
		log.Printf("store: created bucket %q", bucket)
	}

	return &MinIO{client: client, bucket: bucket}, nil
}

// Upload stores the submission binary in MinIO and returns the object key.
// Key format: submissions/<submissionID>/<filename>
func (m *MinIO) Upload(ctx context.Context, submissionID string, filename string, reader io.Reader, size int64) (string, error) {
	key := fmt.Sprintf("submissions/%s/%s", submissionID, filename)

	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", fmt.Errorf("minio upload: %w", err)
	}

	return key, nil
}
