package inference

import (
	"context"

	schemas "github.com/cannonball10/foundation/schemas/inference"
)

// TextInferenceConnector abstracts text completion across providers.
type TextInferenceConnector interface {
	Complete(ctx context.Context, req *schemas.CompletionRequest) (*schemas.CompletionResponse, error)
	CompleteStream(ctx context.Context, req *schemas.CompletionRequest) (*CompletionStream, error)
}

// CompletionStream is an iterator over streaming completion events.
// Usage:
//
//	stream, err := connector.CompleteStream(ctx, req)
//	if err != nil { ... }
//	defer stream.Close()
//	for stream.Next() {
//	    event := stream.Event()
//	    // process event
//	}
//	if err := stream.Err(); err != nil { ... }
type CompletionStream struct {
	nextFn  func() bool
	eventFn func() schemas.StreamEvent
	errFn   func() error
	closeFn func() error
}

// NewCompletionStream creates a stream from callback functions.
func NewCompletionStream(
	nextFn func() bool,
	eventFn func() schemas.StreamEvent,
	errFn func() error,
	closeFn func() error,
) *CompletionStream {
	return &CompletionStream{
		nextFn:  nextFn,
		eventFn: eventFn,
		errFn:   errFn,
		closeFn: closeFn,
	}
}

// Next advances to the next event. Returns false when the stream is exhausted or an error occurred.
func (s *CompletionStream) Next() bool {
	return s.nextFn()
}

// Event returns the current stream event. Only valid after Next() returns true.
func (s *CompletionStream) Event() schemas.StreamEvent {
	return s.eventFn()
}

// Err returns the first error encountered during iteration.
func (s *CompletionStream) Err() error {
	return s.errFn()
}

// Close releases resources associated with the stream.
func (s *CompletionStream) Close() error {
	return s.closeFn()
}
