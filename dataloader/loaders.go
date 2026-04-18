package dataloader

import (
	"context"

	"github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/models"
	dbSchema "github.com/cannonball10/foundation/schemas/database"
)

type contextKey struct{}

// Loaders holds all model-specific dataloaders for a single request.
type Loaders struct {
	UserLoader *Loader[*models.User]
}

// NewLoaders creates a new Loaders instance. Call this once per request.
func NewLoaders(db database.DatabaseConnector) *Loaders {
	return &Loaders{
		UserLoader: NewLoader(db,
			func(k CompositeKey) dbSchema.Key { return models.UserKeys.Key(k.Part1) },
			func(u *models.User) CompositeKey { return CompositeKey{u.UserID, u.UserID} },
		),
	}
}

// WithLoaders returns a new context carrying the given Loaders.
func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, contextKey{}, loaders)
}

// FromContext retrieves Loaders from the context. Panics if not present.
func FromContext(ctx context.Context) *Loaders {
	return ctx.Value(contextKey{}).(*Loaders)
}
