package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"github.com/cannonball10/foundation/schemas/storage"
)

type StorageConnector interface {
	// Upload writes a blob to storage under the given key.
	Upload(ctx context.Context, key string, body io.Reader, opts *storage.UploadOptions) (*storage.UploadResult, error)

	// GeneratePresignedURL creates a time-limited URL for GET or PUT of the given key.
	GeneratePresignedURL(ctx context.Context, key string, opts *storage.PresignOptions) (string, error)

	// GetObject retrieves an object from storage. The caller is responsible for closing the reader.
	GetObject(ctx context.Context, key string, opts *storage.GetOptions) (io.ReadCloser, *storage.GetResult, error)
}

// UploadBytes is a convenience helper for uploading an in-memory buffer.
func UploadBytes(ctx context.Context, c StorageConnector, key string, b []byte, opts *storage.UploadOptions) (*storage.UploadResult, error) {
	if c == nil {
		return nil, io.ErrClosedPipe
	}
	return c.Upload(ctx, key, bytes.NewReader(b), opts)
}

func DefaultStorageConnector(ctx context.Context) (StorageConnector, error) {
	bucket := os.Getenv("STORAGE_BUCKET")
	if bucket == "" {
		return nil, errors.New("STORAGE_BUCKET is not set")
	}
	return DefaultS3StorageConnector(ctx, bucket)
}
