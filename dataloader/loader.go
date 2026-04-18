package dataloader

import (
	"context"
	"fmt"

	"github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/models"
	dbSchema "github.com/cannonball10/foundation/schemas/database"
	dl "github.com/graph-gophers/dataloader/v7"
)

// CompositeKey identifies a DynamoDB item by two parts.
// For single-ID models (where PK and SK derive from the same value),
// Part1 and Part2 are equal. For composite models (e.g. Prompt, Link),
// Part1 is the PK ID and Part2 is the SK ID.
type CompositeKey struct {
	Part1 string
	Part2 string
}

// KeyFunc converts a CompositeKey to a DynamoDB key.
type KeyFunc func(key CompositeKey) dbSchema.Key

// MatchKeyFunc extracts a CompositeKey from a fetched model.
type MatchKeyFunc[T models.Model] func(item T) CompositeKey

// Loader wraps a dataloader for a specific model type, batching individual
// Get calls into BulkGet within a single GraphQL request.
type Loader[T models.Model] struct {
	loader *dl.Loader[CompositeKey, T]
}

// Load fetches a single item by its CompositeKey. Concurrent loads within the
// same request are automatically batched.
func (l *Loader[T]) Load(ctx context.Context, key CompositeKey) (T, error) {
	thunk := l.loader.Load(ctx, key)
	return thunk()
}

// LoadMany fetches multiple items by their CompositeKeys.
func (l *Loader[T]) LoadMany(ctx context.Context, keys []CompositeKey) ([]T, []error) {
	thunk := l.loader.LoadMany(ctx, keys)
	return thunk()
}

// LoadByID fetches a single item where PK and SK derive from the same ID.
func (l *Loader[T]) LoadByID(ctx context.Context, id string) (T, error) {
	return l.Load(ctx, CompositeKey{Part1: id, Part2: id})
}

// LoadManyByID fetches multiple items where PK and SK derive from the same ID.
func (l *Loader[T]) LoadManyByID(ctx context.Context, ids []string) ([]T, []error) {
	keys := make([]CompositeKey, len(ids))
	for i, id := range ids {
		keys[i] = CompositeKey{Part1: id, Part2: id}
	}
	return l.LoadMany(ctx, keys)
}

// NewLoader creates a new Loader for the given model type.
//
//   - db: the database connector used for BulkGet calls
//   - keyFn: converts a CompositeKey to a DynamoDB Key
//   - matchFn: extracts the CompositeKey from a fetched model instance
func NewLoader[T models.Model](db database.DatabaseConnector, keyFn KeyFunc, matchFn MatchKeyFunc[T]) *Loader[T] {
	batchFn := func(ctx context.Context, keys []CompositeKey) []*dl.Result[T] {
		results := make([]*dl.Result[T], len(keys))

		// Convert CompositeKeys to DynamoDB keys.
		dbKeys := make([]dbSchema.Key, len(keys))
		for i, k := range keys {
			dbKeys[i] = keyFn(k)
		}

		// Bulk fetch from DynamoDB.
		items, err := db.BulkGet(ctx, nil, dbKeys)
		if err != nil {
			for i := range results {
				results[i] = &dl.Result[T]{Error: fmt.Errorf("bulk get failed: %w", err)}
			}
			return results
		}

		// Build lookup map using matchFn to reorder results.
		lookup := make(map[CompositeKey]T, len(items))
		for _, item := range items {
			typed, ok := item.(T)
			if !ok {
				continue
			}
			lookup[matchFn(typed)] = typed
		}

		// Reorder results to match input key order.
		// Not-found items return zero value with nil error.
		for i, k := range keys {
			if val, found := lookup[k]; found {
				results[i] = &dl.Result[T]{Data: val}
			} else {
				var zero T
				results[i] = &dl.Result[T]{Data: zero}
			}
		}

		return results
	}

	loader := dl.NewBatchedLoader(batchFn,
		dl.WithCache(&dl.NoCache[CompositeKey, T]{}),
	)

	return &Loader[T]{loader: loader}
}
