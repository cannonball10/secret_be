package inference

import (
	"context"
	"log/slog"
)

// InferenceConnector is the composite connector for inference capabilities.
// Currently holds Text for text completions; future modalities (image, TTS)
// would be added as additional fields.
type InferenceConnector struct {
	Text TextInferenceConnector
}

// NewInferenceConnector creates an InferenceConnector with the given text connector.
func NewInferenceConnector(text TextInferenceConnector) InferenceConnector {
	return InferenceConnector{Text: text}
}

// DefaultInferenceConnector creates an InferenceConnector with a default router
// that auto-discovers providers based on available API keys.
func DefaultInferenceConnector(ctx context.Context) (InferenceConnector, error) {
	router, err := DefaultTextInferenceRouter(ctx)
	if err != nil {
		return InferenceConnector{}, err
	}
	return NewInferenceConnector(router), nil
}

// Close releases resources held by sub-connectors that implement io.Closer.
func (c InferenceConnector) Close() error {
	type closer interface {
		Close() error
	}
	if cl, ok := c.Text.(closer); ok {
		if err := cl.Close(); err != nil {
			slog.Error("inference connector close failed", "error", err)
			return err
		}
	}
	return nil
}
