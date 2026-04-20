package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	audio "github.com/cannonball10/foundation/schemas/audio"
)

// Compile-time check.
var _ TTSConnector = (*ElevenLabs)(nil)

// Default endpoints and model. Overridable via config.
const (
	defaultElevenLabsBaseURL   = "https://api.elevenlabs.io"
	defaultElevenLabsModel     = "eleven_turbo_v2_5"
	defaultElevenLabsChunkSize = 4096 // bytes per stream emission
)

// ElevenLabs is a TTSConnector that talks to the ElevenLabs REST API
// directly (no SDK). The surface we need is tiny: two endpoints,
// JSON in, audio bytes out.
type ElevenLabs struct {
	httpClient   *http.Client
	baseURL      string
	apiKey       string
	defaultVoice string
	defaultModel string
}

// ElevenLabsConfig is the injectable configuration for ElevenLabs.
type ElevenLabsConfig struct {
	// APIKey is the xi-api-key header value. Required.
	APIKey string
	// BaseURL overrides the endpoint host (useful in tests). Optional.
	BaseURL string
	// DefaultVoiceID is used when SynthesisRequest.VoiceID is empty.
	DefaultVoiceID string
	// DefaultModel is used when SynthesisRequest.Model is empty.
	// Defaults to eleven_turbo_v2_5.
	DefaultModel string
	// HTTPClient lets callers inject a custom client (timeouts, middleware).
	HTTPClient *http.Client
}

// NewElevenLabs constructs an ElevenLabs connector. apiKey and a default
// voice are required — ElevenLabs has no "default voice" server-side.
func NewElevenLabs(cfg ElevenLabsConfig) (*ElevenLabs, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("tts: ElevenLabs APIKey is required")
	}
	if cfg.DefaultVoiceID == "" {
		return nil, errors.New("tts: ElevenLabs DefaultVoiceID is required (set ELEVENLABS_VOICE_ID or pass per-request)")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultElevenLabsBaseURL
	}
	model := cfg.DefaultModel
	if model == "" {
		model = defaultElevenLabsModel
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &ElevenLabs{
		httpClient:   client,
		baseURL:      base,
		apiKey:       cfg.APIKey,
		defaultVoice: cfg.DefaultVoiceID,
		defaultModel: model,
	}, nil
}

// DefaultElevenLabsConnector reads ELEVENLABS_API_KEY, ELEVENLABS_VOICE_ID,
// and optionally ELEVENLABS_MODEL_ID from the environment.
func DefaultElevenLabsConnector(_ context.Context) (*ElevenLabs, error) {
	return NewElevenLabs(ElevenLabsConfig{
		APIKey:         os.Getenv("ELEVENLABS_API_KEY"),
		DefaultVoiceID: os.Getenv("ELEVENLABS_VOICE_ID"),
		DefaultModel:   os.Getenv("ELEVENLABS_MODEL_ID"),
	})
}

// elevenBody is the JSON payload shape for both TTS endpoints.
type elevenBody struct {
	Text          string         `json:"text"`
	ModelID       string         `json:"model_id,omitempty"`
	LanguageCode  string         `json:"language_code,omitempty"`
	VoiceSettings *voiceSettings `json:"voice_settings,omitempty"`
}

type voiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style,omitempty"`
}

// voiceSettingsFrom returns voice_settings iff at least one value is non-zero.
// Omitting the block entirely lets the server apply its configured defaults.
func voiceSettingsFrom(req *SynthesisRequest) *voiceSettings {
	if req.Stability == 0 && req.Similarity == 0 && req.Style == 0 {
		return nil
	}
	return &voiceSettings{
		Stability:       req.Stability,
		SimilarityBoost: req.Similarity,
		Style:           req.Style,
	}
}

// outputFormatParam maps our AudioFormat to ElevenLabs' output_format query
// value. Returns "" to signal "let the server pick the default", which
// will be mp3_44100_128.
func outputFormatParam(f audio.AudioFormat, sampleRate int) string {
	switch f {
	case "", audio.FormatMP3:
		// MP3 rates: 22050/32, 44100/32, 44100/64, 44100/96, 44100/128, 44100/192.
		// We map the common cases; anything else falls back to 44100/128.
		switch sampleRate {
		case audio.SampleRate22kHz:
			return "mp3_22050_32"
		default:
			return "mp3_44100_128"
		}
	case audio.FormatPCM16:
		switch sampleRate {
		case audio.SampleRate8kHz:
			return "pcm_8000"
		case audio.SampleRate16kHz:
			return "pcm_16000"
		case audio.SampleRate22kHz:
			return "pcm_22050"
		case audio.SampleRate24kHz:
			return "pcm_24000"
		case audio.SampleRate44kHz:
			return "pcm_44100"
		default:
			return "pcm_24000"
		}
	case audio.FormatULaw:
		return "ulaw_8000"
	case audio.FormatOpus:
		return "opus_48000_96"
	default:
		return ""
	}
}

// buildRequest constructs a POST request with body + headers. The query
// string carries output_format; the body carries text/model/settings.
func (e *ElevenLabs) buildRequest(ctx context.Context, req *SynthesisRequest, stream bool) (*http.Request, error) {
	if req == nil || req.Text == "" {
		return nil, errors.New("tts: SynthesisRequest.Text is required")
	}
	voice := req.VoiceID
	if voice == "" {
		voice = e.defaultVoice
	}
	model := req.Model
	if model == "" {
		model = e.defaultModel
	}

	body := elevenBody{
		Text:          req.Text,
		ModelID:       model,
		LanguageCode:  req.LanguageCode,
		VoiceSettings: voiceSettingsFrom(req),
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("tts: marshal request: %w", err)
	}

	path := fmt.Sprintf("/v1/text-to-speech/%s", voice)
	if stream {
		path += "/stream"
	}
	url := e.baseURL + path
	if outFmt := outputFormatParam(req.Format, req.SampleRate); outFmt != "" {
		url += "?output_format=" + outFmt
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("tts: build request: %w", err)
	}
	httpReq.Header.Set("xi-api-key", e.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "audio/*")
	return httpReq, nil
}

// Synthesize implements TTSConnector.
func (e *ElevenLabs) Synthesize(ctx context.Context, req *SynthesisRequest) (*SynthesisResult, error) {
	httpReq, err := e.buildRequest(ctx, req, false)
	if err != nil {
		return nil, err
	}
	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		slog.ErrorContext(ctx, "elevenlabs synthesize failed", "error", err)
		return nil, fmt.Errorf("tts: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, readElevenError(resp)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts: read body: %w", err)
	}
	format := req.Format
	if format == "" {
		format = audio.FormatMP3
	}
	return &SynthesisResult{
		Audio:      data,
		Format:     format,
		SampleRate: req.SampleRate,
	}, nil
}

// SynthesizeStream implements TTSConnector. The HTTP response body is a
// chunked stream of raw audio bytes (no framing); we slice it into fixed-size
// AudioChunks as they arrive and feed them to the caller through a channel.
func (e *ElevenLabs) SynthesizeStream(ctx context.Context, req *SynthesisRequest) (*AudioStream, error) {
	httpReq, err := e.buildRequest(ctx, req, true)
	if err != nil {
		return nil, err
	}
	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		slog.ErrorContext(ctx, "elevenlabs stream failed", "error", err)
		return nil, fmt.Errorf("tts: http: %w", err)
	}
	if resp.StatusCode >= 400 {
		err := readElevenError(resp)
		resp.Body.Close()
		return nil, err
	}

	// State captured by the iterator closures. Producer goroutine reads
	// the body and pushes chunks onto `chunks`; consumer pulls them in Next().
	chunks := make(chan AudioChunk, 4)
	var (
		streamErr error
		current   AudioChunk
	)
	closed := make(chan struct{})

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		buf := make([]byte, defaultElevenLabsChunkSize)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				out := make([]byte, n)
				copy(out, buf[:n])
				select {
				case chunks <- AudioChunk{Data: out}:
				case <-ctx.Done():
					streamErr = ctx.Err()
					return
				case <-closed:
					return
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					streamErr = fmt.Errorf("tts: read stream: %w", readErr)
				}
				return
			}
		}
	}()

	return NewAudioStream(
		func() bool {
			c, ok := <-chunks
			if !ok {
				return false
			}
			current = c
			return true
		},
		func() AudioChunk { return current },
		func() error { return streamErr },
		func() error {
			select {
			case <-closed:
			default:
				close(closed)
			}
			// Drain the producer so the goroutine exits.
			for range chunks {
			}
			return nil
		},
	), nil
}

// readElevenError parses the non-2xx JSON error envelope ElevenLabs returns.
func readElevenError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Detail any `json:"detail"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Detail != nil {
		return fmt.Errorf("tts: elevenlabs %d: %v", resp.StatusCode, parsed.Detail)
	}
	return fmt.Errorf("tts: elevenlabs %d: %s", resp.StatusCode, bytes.TrimSpace(body))
}
