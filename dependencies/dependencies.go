package dependencies

import (
	"context"
	"os"

	"github.com/cannonball10/foundation/connectors"
	"github.com/cannonball10/foundation/handlers"
	"github.com/hibiken/asynq"
)

type contextKey struct{}

// WithDeps returns a new context carrying the given Dependencies.
func WithDeps(ctx context.Context, deps *Dependencies) context.Context {
	return context.WithValue(ctx, contextKey{}, deps)
}

// FromContext retrieves Dependencies from the context.
// Panics if not present — call only from paths where Dependencies
// are guaranteed (e.g., asynq task handlers via BaseContext).
func FromContext(ctx context.Context) *Dependencies {
	return ctx.Value(contextKey{}).(*Dependencies)
}

type Dependencies struct {
	connectors.Connectors
	handlers.Handlers

	TaskClient *asynq.Client
}

// GetConnectors returns a pointer to the embedded Connectors.
func (d *Dependencies) GetConnectors() *connectors.Connectors {
	return &d.Connectors
}

// GetTaskClient returns the asynq task client.
func (d *Dependencies) GetTaskClient() *asynq.Client {
	return d.TaskClient
}

type DependenciesOptions struct {
	connectors.ConnectorsOptions

	TaskClient *asynq.Client
}

func NewDependencies(opts DependenciesOptions) *Dependencies {
	connectors := connectors.NewConnectors(opts.ConnectorsOptions)
	deps := &Dependencies{
		Connectors: *connectors,
		TaskClient: opts.TaskClient,
	}
	deps.Handlers = *handlers.NewHandlers(deps)
	return deps
}

func DefaultDependencies(ctx context.Context) (*Dependencies, error) {
	connectors, err := connectors.DefaultConnectors(ctx)
	if err != nil {
		return nil, err
	}
	taskClient := asynq.NewClient(asynq.RedisClientOpt{Addr: os.Getenv("REDIS_ADDR")})
	deps := &Dependencies{
		Connectors: *connectors,
		TaskClient: taskClient,
	}
	deps.Handlers = *handlers.NewHandlers(deps)
	return deps, nil
}
