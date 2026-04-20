// Package stt abstracts speech-to-text providers behind a common
// interface. Two calling shapes are supported:
//
//   - Transcribe: single-shot. Hand in a full audio buffer, get the
//     final transcript back.
//   - TranscribeStream: duplex. Push audio chunks in with Send, signal
//     end-of-audio with CloseSend, then iterate TranscriptEvents.
//
// Concrete providers live in sibling files (see openai.go).
package stt

import (
	"context"

	audio "github.com/cannonball10/foundation/schemas/audio"
)

// STTConnector is the provider-agnostic speech-to-text interface.
type STTConnector interface {
	// Transcribe runs a single-shot transcription on the full audio in req.
	Transcribe(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResult, error)

	// TranscribeStream opens a streaming transcription session. The
	// caller pushes audio bytes via Send() and pulls events via Next().
	// Ownership: caller must Close() the returned stream.
	TranscribeStream(ctx context.Context, params *StreamParams) (*TranscriptionStream, error)
}

// TranscriptionRequest is the input to Transcribe.
type TranscriptionRequest struct {
	// Audio is the full encoded payload. Required.
	Audio []byte

	// Format identifies the encoding of Audio. Required for providers
	// that do not sniff it from magic bytes.
	Format audio.AudioFormat

	// SampleRate in Hz; required for raw PCM, optional otherwise.
	SampleRate int

	// Language is an optional BCP-47 hint (e.g. "en"). Empty = auto.
	Language string

	// Prompt is an optional context/style hint. OpenAI uses it to bias
	// vocabulary; providers that do not support it ignore.
	Prompt string

	// Model selects the provider model. Empty = provider default.
	Model string

	// Temperature is the sampling temperature, [0,1]. Zero = default.
	Temperature float64
}

// TranscriptionResult is the output of a single-shot Transcribe.
type TranscriptionResult struct {
	// Text is the final full transcript.
	Text string
	// Language is the detected (or echoed) BCP-47 code, if the provider reports it.
	Language string
	// Segments may be non-nil when the provider returns word/utterance timings.
	Segments []TranscriptSegment
}

// TranscriptSegment is one timed span of transcript. Only populated by
// providers that support verbose responses.
type TranscriptSegment struct {
	Text  string
	Start float64 // seconds
	End   float64 // seconds
}

// StreamParams is the init payload for TranscribeStream. Streaming does
// not carry audio up-front; audio arrives via Send() calls.
type StreamParams struct {
	// Format of the audio chunks the caller will Send.
	Format audio.AudioFormat
	// SampleRate in Hz; required for PCM.
	SampleRate int
	// Language is an optional BCP-47 hint.
	Language string
	// Prompt is an optional vocabulary bias.
	Prompt string
	// Model selects the provider model. Empty = provider default.
	Model string
}

// TranscriptEvent is one message on the output side of a streaming session.
type TranscriptEvent struct {
	// Delta is the newly-transcribed text since the last event.
	// May be empty when Type != EventDelta.
	Delta string

	// Text is the accumulated transcript so far (or the final transcript
	// on EventFinal).
	Text string

	// Type classifies the event.
	Type EventType
}

// EventType classifies TranscriptEvents.
type EventType string

const (
	// EventDelta carries a partial token/word update.
	EventDelta EventType = "delta"
	// EventFinal marks the final full transcript; no further events follow.
	EventFinal EventType = "final"
)

// TranscriptionStream is a duplex handle for streaming STT:
//
//	stream, err := conn.TranscribeStream(ctx, params)
//	if err != nil { ... }
//	defer stream.Close()
//
//	// producer side — typically a separate goroutine reading a mic/file:
//	go func() {
//	    for chunk := range mic { stream.Send(chunk) }
//	    stream.CloseSend()
//	}()
//
//	// consumer side:
//	for stream.Next() {
//	    ev := stream.Event()
//	    switch ev.Type { case EventDelta: ... case EventFinal: ... }
//	}
//	if err := stream.Err(); err != nil { ... }
//
// Send and CloseSend are safe to call from a different goroutine than
// Next/Event/Err.
type TranscriptionStream struct {
	sendFn      func([]byte) error
	closeSendFn func() error
	nextFn      func() bool
	eventFn     func() TranscriptEvent
	errFn       func() error
	closeFn     func() error
}

// NewTranscriptionStream wires the six callbacks into a stream handle.
// Providers own the goroutines, buffering, and cancellation behind the
// callbacks.
func NewTranscriptionStream(
	sendFn func([]byte) error,
	closeSendFn func() error,
	nextFn func() bool,
	eventFn func() TranscriptEvent,
	errFn func() error,
	closeFn func() error,
) *TranscriptionStream {
	return &TranscriptionStream{
		sendFn:      sendFn,
		closeSendFn: closeSendFn,
		nextFn:      nextFn,
		eventFn:     eventFn,
		errFn:       errFn,
		closeFn:     closeFn,
	}
}

// Send pushes one audio chunk into the stream. The encoding must match
// the StreamParams.Format passed to TranscribeStream.
func (s *TranscriptionStream) Send(chunk []byte) error { return s.sendFn(chunk) }

// CloseSend signals that no more audio will be sent. Providers that
// require a full buffer before transcribing will begin processing now.
// After CloseSend it is safe to Next() until the stream drains.
func (s *TranscriptionStream) CloseSend() error { return s.closeSendFn() }

// Next advances to the next event. Returns false on drain or error.
func (s *TranscriptionStream) Next() bool { return s.nextFn() }

// Event returns the current event. Only valid after Next() returns true.
func (s *TranscriptionStream) Event() TranscriptEvent { return s.eventFn() }

// Err returns the first error encountered, if any.
func (s *TranscriptionStream) Err() error { return s.errFn() }

// Close releases resources backing the stream. Always safe to call.
func (s *TranscriptionStream) Close() error { return s.closeFn() }

// DefaultSTTConnector returns the default provider. Currently OpenAI.
func DefaultSTTConnector(ctx context.Context) (STTConnector, error) {
	return DefaultOpenAIConnector(ctx)
}
