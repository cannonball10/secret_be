package secret

import "context"

type SecretConnector interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any) error
	Update(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) error
}

func DefaultSecretConnector(ctx context.Context) (SecretConnector, error) {
	return DefaultSecretManagerConnector(ctx)
}
