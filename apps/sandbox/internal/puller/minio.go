package puller

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioPuller struct {
	client *minio.Client
	bucket string
	tmpDir string
}

func NewMinioPuller(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioPuller, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}

	// each sandbox run gets its own subdirectory under this temp root
	tmpDir, err := os.MkdirTemp("", "sandbox-binaries-*")
	if err != nil {
		return nil, fmt.Errorf("create tmp dir: %w", err)
	}
	log.Printf("puller: using tmp dir %s", tmpDir)

	return &MinioPuller{client: client, bucket: bucket, tmpDir: tmpDir}, nil
}

// Pull downloads the submitted binary from minio to a local temp file so docker can copy it in
func (p *MinioPuller) Pull(ctx context.Context, submissionID, s3Key string) (string, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, s3Key, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("minio get %s: %w", s3Key, err)
	}
	defer obj.Close()

	// Create a local file for the binary
	localDir := filepath.Join(p.tmpDir, submissionID)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", localDir, err)
	}

	localPath := filepath.Join(localDir, "binary")
	f, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, obj)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", s3Key, err)
	}

	if err := os.Chmod(localPath, 0755); err != nil { // needs to be executable inside the container
		return "", fmt.Errorf("chmod binary: %w", err)
	}

	log.Printf("puller: downloaded %s → %s (%d bytes)", s3Key, localPath, n)
	return localPath, nil
}

func (p *MinioPuller) Cleanup(submissionID string) {
	dir := filepath.Join(p.tmpDir, submissionID)
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("puller: cleanup warning for %s: %v", submissionID, err)
	}
}

func (p *MinioPuller) Close() error {
	return os.RemoveAll(p.tmpDir)
}
