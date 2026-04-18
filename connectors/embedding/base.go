package embedding

import (
	"context"
)

// EmbeddingConnector is an abstract embedding interface that can be extended for concrete implementations like OpenAI, etc.
type EmbeddingConnector interface {
	Embed(ctx context.Context, payload any) ([]float64, error)
	EmbedBatch(ctx context.Context, payloads []any) ([][]float64, error)
}

func DefaultEmbeddingConnector(ctx context.Context) (EmbeddingConnector, error) {
	return DefaultOpenAIEmbeddingConnector(ctx)
}
