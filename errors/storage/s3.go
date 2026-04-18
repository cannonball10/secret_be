package storage

import "errors"

var (
	ErrStorageBucketEmpty          = errors.New("storage bucket is empty")
	ErrStorageKeyEmpty             = errors.New("storage key is empty")
	ErrStorageClientUninitialized  = errors.New("s3 client is not initialized")
	ErrStorageUploadBodyNil        = errors.New("upload body is nil")
	ErrStorageUnsupportedOperation = errors.New("unsupported presign operation")
)
