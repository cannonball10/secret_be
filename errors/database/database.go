package database

import "errors"

var ErrDatabaseCollectionMissing = errors.New("database collection name is required")
var ErrDatabaseClientUninitialized = errors.New("database client is not initialized")
var ErrDatabaseTransactionWriteExceededRetries = errors.New("database transaction write exceeded retries")
var ErrDatabaseBatchWriteExceededRetries = errors.New("database batch write exceeded retries")
var ErrDatabaseBatchGetExceededRetries = errors.New("database batch get exceeded retries")
var ErrDatabaseTransactionWriteExceededLimit = errors.New("database transaction write exceeded limit")
var ErrDatabaseBatchWriteExceededLimit = errors.New("database batch write exceeded limit")
var ErrDatabaseBatchGetExceededLimit = errors.New("database batch get exceeded limit")
