package dynamo

import (
	"context"
	"errors"
	"os"
	"time"
)

var ErrDynamoCollectionMissing = errors.New("dynamo collection name is required")

// chunkSlice splits a slice into chunks of the specified size.
func ChunkSlice[T any](items []T, limit int) [][]T {
	if limit <= 0 || len(items) <= limit {
		return [][]T{items}
	}

	chunks := make([][]T, 0, (len(items)+limit-1)/limit)
	for start := 0; start < len(items); start += limit {
		end := start + limit
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

// sleepWithContext sleeps for the specified duration, but returns early if context is cancelled.
func SleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// resolveCollection resolves the table name from collection parameter or environment variable.
func ResolveCollection(collection *string) (string, error) {
	if collection != nil && *collection != "" {
		return *collection, nil
	}
	table := os.Getenv("FOUNDATION_DATABASE_TABLE")
	if table == "" {
		return "", ErrDynamoCollectionMissing
	}
	return table, nil
}
