package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	storageErrors "github.com/cannonball10/foundation/errors/storage"
	"github.com/cannonball10/foundation/schemas/storage"
)

var _ StorageConnector = (*S3Storage)(nil)

const defaultPresignExpires = 15 * time.Minute

// S3Storage implements StorageConnector using AWS S3 (AWS SDK v2).
// It can be constructed with an eager client or lazily via a factory.
type S3Storage struct {
	bucket string

	client        *s3.Client
	clientOnce    sync.Once
	clientFactory func() *s3.Client
}

// NewS3Storage returns a connector with an eagerly provided client.
func NewS3Storage(client *s3.Client, bucket string) *S3Storage {
	return newS3Storage(client, nil, bucket)
}

// NewS3StorageLazy returns a connector that initializes the client on first use.
func NewS3StorageLazy(factory func() *s3.Client, bucket string) *S3Storage {
	return newS3Storage(nil, factory, bucket)
}

func DefaultS3StorageConnector(ctx context.Context, bucket string) (*S3Storage, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config LoadDefaultConfig: %w", err)
	}
	return NewS3Storage(s3.NewFromConfig(cfg), bucket), nil
}

func newS3Storage(client *s3.Client, factory func() *s3.Client, bucket string) *S3Storage {
	return &S3Storage{
		bucket:        bucket,
		client:        client,
		clientFactory: factory,
	}
}

func (s *S3Storage) Upload(ctx context.Context, key string, body io.Reader, opts *storage.UploadOptions) (*storage.UploadResult, error) {
	if s.bucket == "" && opts.Bucket == nil {
		return nil, storageErrors.ErrStorageBucketEmpty
	}
	bucket := s.bucket
	if opts.Bucket != nil {
		bucket = *opts.Bucket
	}
	if key == "" {
		return nil, storageErrors.ErrStorageKeyEmpty
	}
	if body == nil {
		return nil, storageErrors.ErrStorageUploadBodyNil
	}
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}

	in := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if opts != nil {
		if opts.ContentType != "" {
			in.ContentType = aws.String(opts.ContentType)
		}
		if opts.CacheControl != "" {
			in.CacheControl = aws.String(opts.CacheControl)
		}
		if opts.Metadata != nil {
			in.Metadata = opts.Metadata
		}
	}

	uploader := manager.NewUploader(client)
	out, err := uploader.Upload(ctx, in)
	if err != nil {
		slog.ErrorContext(ctx, "s3 Upload failed", "bucket", bucket, "key", key, "error", err)
		return nil, fmt.Errorf("s3 upload %q: %w", key, err)
	}

	res := &storage.UploadResult{
		Bucket: bucket,
		Key:    key,
	}
	if out != nil {
		if out.ETag != nil {
			res.ETag = aws.ToString(out.ETag)
		}
		if out.VersionID != nil {
			res.VersionID = aws.ToString(out.VersionID)
		}
		// Prefer response key if present.
		if out.Key != nil && aws.ToString(out.Key) != "" {
			res.Key = aws.ToString(out.Key)
		}
	}
	return res, nil
}

func (s *S3Storage) GeneratePresignedURL(ctx context.Context, key string, opts *storage.PresignOptions) (string, error) {
	if s.bucket == "" && opts.Bucket == nil {
		return "", storageErrors.ErrStorageBucketEmpty
	}
	bucket := s.bucket
	if opts.Bucket != nil {
		bucket = *opts.Bucket
	}
	client, err := s.getClient()
	if err != nil {
		return "", err
	}

	op := storage.PresignGetObject
	expires := defaultPresignExpires
	if opts != nil {
		if opts.Operation != "" {
			op = opts.Operation
		}
		if opts.Expires > 0 {
			expires = opts.Expires
		}
	}

	presign := s3.NewPresignClient(client)
	switch op {
	case storage.PresignGetObject:
		req, err := presign.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}, func(o *s3.PresignOptions) {
			o.Expires = expires
		})
		if err != nil {
			slog.ErrorContext(ctx, "s3 PresignGet failed", "bucket", bucket, "key", key, "error", err)
			return "", fmt.Errorf("s3 presign GET %q: %w", key, err)
		}
		return req.URL, nil
	case storage.PresignPutObject:
		req, err := presign.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}, func(o *s3.PresignOptions) {
			o.Expires = expires
		})
		if err != nil {
			slog.ErrorContext(ctx, "s3 PresignPut failed", "bucket", bucket, "key", key, "error", err)
			return "", fmt.Errorf("s3 presign PUT %q: %w", key, err)
		}
		return req.URL, nil
	default:
		return "", storageErrors.ErrStorageUnsupportedOperation
	}
}

func (s *S3Storage) GetObject(ctx context.Context, key string, opts *storage.GetOptions) (io.ReadCloser, *storage.GetResult, error) {
	bucket := s.bucket
	if opts != nil && opts.Bucket != nil {
		bucket = *opts.Bucket
	}
	if bucket == "" {
		return nil, nil, storageErrors.ErrStorageBucketEmpty
	}
	if key == "" {
		return nil, nil, storageErrors.ErrStorageKeyEmpty
	}
	client, err := s.getClient()
	if err != nil {
		return nil, nil, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if opts != nil && opts.VersionID != "" {
		input.VersionId = aws.String(opts.VersionID)
	}

	out, err := client.GetObject(ctx, input)
	if err != nil {
		slog.ErrorContext(ctx, "s3 GetObject failed", "bucket", bucket, "key", key, "error", err)
		return nil, nil, fmt.Errorf("s3 get %q: %w", key, err)
	}

	result := &storage.GetResult{}
	if out.ContentType != nil {
		result.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		result.ContentLength = *out.ContentLength
	}
	if out.ETag != nil {
		result.ETag = *out.ETag
	}
	if out.LastModified != nil {
		result.LastModified = *out.LastModified
	}

	return out.Body, result, nil
}

func (s *S3Storage) getClient() (*s3.Client, error) {
	s.clientOnce.Do(func() {
		if s.client == nil && s.clientFactory != nil {
			s.client = s.clientFactory()
		}
	})
	if s.client == nil {
		return nil, storageErrors.ErrStorageClientUninitialized
	}
	return s.client, nil
}
