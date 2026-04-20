// Package tts abstracts text-to-speech providers behind a common
// interface. Two calling shapes are supported:
//
//   - Synthesize: single-shot. Hand in text, get the full audio back.
//   - SynthesizeStream: chunked. Iterate the returned AudioStream and
//     forward chunks to the speaker (or a WebSocket) as they arrive.
//
// Concrete providers live in sibling files (see elevenlabs.go).
package tts

import (
	"context"

	audio "github.com/cannonball10/foundation/schemas/audio"
)

// TTSConnector is the provider-agnostic text-to-speech interface.
type TTSConnector interface {
	// Synthesize renders the full request to audio in one call.
	Synthesize(ctx context.Context, req *SynthesisRequest) (*SynthesisResult, error)
	// SynthesizeStream returns an iterator over audio chunks produced
	// progressively by the provider. Caller is responsible for Close().
	SynthesizeStream(ctx context.Context, req *SynthesisRequest) (*AudioStream, error)
}

// SynthesisRequest is the input to both Synthesize and SynthesizeStream.
// Every field except Text is optional; zero values mean "use provider default".
type SynthesisRequest struct {
	// Text is the utterance to speak. Required.
	Text string

	// VoiceID selects the provider-specific voice. For ElevenLabs this
	// is a voice UUID from the user's account. Empty means provider default.
	VoiceID string

	// Model selects the provider-specific model (e.g. eleven_turbo_v2_5).
	// Empty means provider default.
	Model string

	// Format is the desired output audio format. Empty means provider
	// default, which is usually FormatMP3.
	Format audio.AudioFormat

	// SampleRate is the desired output sample rate in Hz. Zero means
	// provider default. Ignored if the format has a baked-in rate.
	SampleRate int

	// Stability, Similarity, Style are ElevenLabs voice_settings knobs
	// in the range [0,1]. Zero = use server-side defaults. Other
	// providers ignore them.
	Stability  float64
	Similarity float64
	Style      float64

	// LanguageCode is an optional BCP-47 tag (e.g. "en", "es-MX").
	// Providers that auto-detect will ignore this.
	LanguageCode string
}

// SynthesisResult is the output of a single-shot Synthesize call.
type SynthesisResult struct {
	// Audio is the full encoded audio payload.
	Audio []byte
	// Format reports the actual encoding of Audio. May differ from
	// the request if the provider substituted a format.
	Format audio.AudioFormat
	// SampleRate reports the actual sample rate; zero if not meaningful
	// for Format (e.g. when MP3's rate is baked into the bitstream).
	SampleRate int
}

// AudioChunk is one frame of streaming audio.
type AudioChunk struct {
	// Data is a contiguous byte slice. Interpretation depends on Format.
	// For MP3 this is a decodable substream; for PCM it's raw samples.
	Data []byte
	// Final is true on the last chunk in the stream.
	Final bool
}

// AudioStream is a pull-style iterator over audio chunks. Shape mirrors
// inference.CompletionStream so callers have the same ergonomics across
// modalities:
//
//	stream, err := conn.SynthesizeStream(ctx, req)
//	if err != nil { ... }
//	defer stream.Close()
//	for stream.Next() {
//	    chunk := stream.Chunk()
//	    // write chunk.Data to the speaker / websocket / disk
//	}
//	if err := stream.Err(); err != nil { ... }
type AudioStream struct {
	nextFn  func() bool
	chunkFn func() AudioChunk
	errFn   func() error
	closeFn func() error
}

// NewAudioStream wires four callbacks into an AudioStream. Providers use
// this to hide their goroutines/channels behind a clean iterator.
func NewAudioStream(
	nextFn func() bool,
	chunkFn func() AudioChunk,
	errFn func() error,
	closeFn func() error,
) *AudioStream {
	return &AudioStream{
		nextFn:  nextFn,
		chunkFn: chunkFn,
		errFn:   errFn,
		closeFn: closeFn,
	}
}

// Next advances to the next chunk. Returns false on exhaustion or error.
func (s *AudioStream) Next() bool { return s.nextFn() }

// Chunk returns the current chunk. Only valid after Next() returns true.
func (s *AudioStream) Chunk() AudioChunk { return s.chunkFn() }

// Err returns the first error encountered during iteration, if any.
func (s *AudioStream) Err() error { return s.errFn() }

// Close releases resources backing the stream. Always safe to call.
func (s *AudioStream) Close() error { return s.closeFn() }

// DefaultTTSConnector returns the default provider. Currently ElevenLabs.
func DefaultTTSConnector(ctx context.Context) (TTSConnector, error) {
	return DefaultElevenLabsConnector(ctx)
}
