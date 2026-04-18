package testmocks

import (
	"context"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
)

// DatabaseConnector is a configurable mock for database.DatabaseConnector.
// Set the function fields you need; unset methods return zero values.
type DatabaseConnector struct {
	GetFunc        func(ctx context.Context, collection *string, key database.Key) (models.Model, error)
	QueryFunc      func(ctx context.Context, collection *string, input database.QueryInput, opts database.QueryOptions) (*database.QueryOutput, error)
	UpsertFunc     func(ctx context.Context, collection *string, item models.Model) error
	DeleteFunc     func(ctx context.Context, collection *string, key database.Key) error
	BulkGetFunc    func(ctx context.Context, collection *string, keys []database.Key) ([]models.Model, error)
	BulkUpsertFunc func(ctx context.Context, collection *string, items []models.Model, opts *database.BulkOptions) error
}

func (m *DatabaseConnector) Get(ctx context.Context, collection *string, key database.Key) (models.Model, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, collection, key)
	}
	return nil, nil
}

func (m *DatabaseConnector) Query(ctx context.Context, collection *string, input database.QueryInput, opts database.QueryOptions) (*database.QueryOutput, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, collection, input, opts)
	}
	return nil, nil
}

func (m *DatabaseConnector) Upsert(ctx context.Context, collection *string, item models.Model) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, collection, item)
	}
	return nil
}

func (m *DatabaseConnector) Delete(ctx context.Context, collection *string, key database.Key) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, collection, key)
	}
	return nil
}

func (m *DatabaseConnector) BulkGet(ctx context.Context, collection *string, keys []database.Key) ([]models.Model, error) {
	if m.BulkGetFunc != nil {
		return m.BulkGetFunc(ctx, collection, keys)
	}
	return nil, nil
}

func (m *DatabaseConnector) BulkUpsert(ctx context.Context, collection *string, items []models.Model, opts *database.BulkOptions) error {
	if m.BulkUpsertFunc != nil {
		return m.BulkUpsertFunc(ctx, collection, items, opts)
	}
	return nil
}

func (m *DatabaseConnector) BulkDelete(ctx context.Context, collection *string, keys []database.Key, opts *database.BulkOptions) error {
	return nil
}

func (m *DatabaseConnector) ListCollections(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *DatabaseConnector) CreateCollection(ctx context.Context, collection *string) error {
	return nil
}

func (m *DatabaseConnector) DeleteCollection(ctx context.Context, collection *string) error {
	return nil
}
