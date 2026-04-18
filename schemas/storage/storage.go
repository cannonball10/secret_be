package storage

import "time"

// UploadOptions configures upload behavior. Nil values use defaults.
type UploadOptions struct {
	Bucket       *string
	ContentType  string
	CacheControl string
	Metadata     map[string]string
}

type UploadResult struct {
	Bucket    string
	Key       string
	ETag      string
	VersionID string
}

type PresignOperation string

const (
	PresignGetObject PresignOperation = "GET_OBJECT"
	PresignPutObject PresignOperation = "PUT_OBJECT"
)

// PresignOptions configures presigned URL generation. Nil values use defaults.
type PresignOptions struct {
	Bucket    *string
	Operation PresignOperation
	Expires   time.Duration
}

// GetOptions configures object retrieval behavior. Nil values use defaults.
type GetOptions struct {
	Bucket    *string
	VersionID string
}

// GetResult contains metadata about a retrieved object.
type GetResult struct {
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  time.Time
}
