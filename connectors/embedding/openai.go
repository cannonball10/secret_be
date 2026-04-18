package embedding

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openai/openai-go/v3"
)

const defaultModel = openai.EmbeddingModelTextEmbedding3Small

// Ensure *OpenAIEmbedding implements EmbeddingConnector.
var _ EmbeddingConnector = (*OpenAIEmbedding)(nil)

// OpenAIEmbedding implements EmbeddingConnector using the OpenAI embeddings API.
type OpenAIEmbedding struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAIEmbedding returns an OpenAIEmbedding that uses the given client and model.
// If model is nil, text-embedding-3-small is used.
func NewOpenAIEmbedding(client *openai.Client, model *openai.EmbeddingModel) *OpenAIEmbedding {
	var usedModel openai.EmbeddingModel
	if model == nil {
		usedModel = defaultModel
	} else {
		usedModel = *model
	}
	return &OpenAIEmbedding{client: client, model: usedModel}
}

func DefaultOpenAIEmbeddingConnector(ctx context.Context) (*OpenAIEmbedding, error) {
	client := openai.NewClient(openai.DefaultClientOptions()...)
	return NewOpenAIEmbedding(&client, nil), nil
}

func payloadToStr(payload any) (string, error) {
	if payload == nil {
		return "", errors.New("embedding payload cannot be nil")
	}
	switch v := payload.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprint(v), nil
	}
}

// Embed implements EmbeddingConnector.
func (o *OpenAIEmbedding) Embed(ctx context.Context, payload any) ([]float64, error) {
	s, err := payloadToStr(payload)
	if err != nil {
		return nil, err
	}
	res, err := o.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(s),
		},
		Model:          openai.EmbeddingModel(o.model),
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	})
	if err != nil {
		slog.ErrorContext(ctx, "openai Embed failed", "input_count", 1, "error", err)
		return nil, fmt.Errorf("embedding: %w", err)
	}
	if len(res.Data) == 0 {
		return nil, errors.New("embedding: empty response from API")
	}
	return res.Data[0].Embedding, nil
}

// EmbedBatch implements EmbeddingConnector.
func (o *OpenAIEmbedding) EmbedBatch(ctx context.Context, payloads []any) ([][]float64, error) {
	if len(payloads) == 0 {
		return [][]float64{}, nil
	}
	inputs := make([]string, 0, len(payloads))
	for i, p := range payloads {
		s, err := payloadToStr(p)
		if err != nil {
			return nil, fmt.Errorf("embedding batch: payload at index %d: %w", i, err)
		}
		inputs = append(inputs, s)
	}
	res, err := o.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: inputs,
		},
		Model:          openai.EmbeddingModel(o.model),
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	})
	if err != nil {
		slog.ErrorContext(ctx, "openai EmbedBatch failed", "input_count", len(inputs), "error", err)
		return nil, fmt.Errorf("embedding batch: %w", err)
	}
	out := make([][]float64, len(res.Data))
	for i := range res.Data {
		out[i] = res.Data[i].Embedding
	}
	return out, nil
}
