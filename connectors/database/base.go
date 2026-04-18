package database

import (
	"context"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
)

// DatabaseConnector is a generic interface for non-relational data stores.
type DatabaseConnector interface {
	Get(ctx context.Context, collection *string, key database.Key) (models.Model, error)
	Query(ctx context.Context, collection *string, input database.QueryInput, opts database.QueryOptions) (*database.QueryOutput, error)
	Upsert(ctx context.Context, collection *string, item models.Model) error
	Delete(ctx context.Context, collection *string, key database.Key) error

	BulkGet(ctx context.Context, collection *string, keys []database.Key) ([]models.Model, error)
	BulkUpsert(ctx context.Context, collection *string, items []models.Model, opts *database.BulkOptions) error
	BulkDelete(ctx context.Context, collection *string, keys []database.Key, opts *database.BulkOptions) error

	ListCollections(ctx context.Context) ([]string, error)
	CreateCollection(ctx context.Context, collection *string) error
	DeleteCollection(ctx context.Context, collection *string) error
}

func DefaultDatabaseConnector(ctx context.Context) (DatabaseConnector, error) {
	return DefaultDynamoDatabseConnector(ctx, nil)
}
