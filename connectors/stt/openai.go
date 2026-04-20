package stt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	audio "github.com/cannonball10/foundation/schemas/audio"
)

// Compile-time check.
var _ STTConnector = (*OpenAI)(nil)

const (
	defaultOpenAIBaseURL = "https://api.openai.com"
	// whisper-1 is the stable single-shot model. gpt-4o-transcribe /
	// gpt-4o-mini-transcribe additionally support stream=true on the
	// transcriptions endpoint.
	defaultOpenAIModel       = "whisper-1"
	defaultOpenAIStreamModel = "gpt-4o-mini-transcribe"
)

// OpenAI is an STTConnector that talks to the OpenAI audio/transcriptions
// endpoint directly (no SDK). Single-shot uses the standard multipart
// upload; streaming uses the same endpoint with stream=true and consumes
// the server-sent events response.
type OpenAI struct {
	httpClient   *http.Client
	baseURL      string
	apiKey       string
	model        string
	streamModel  string
}

// OpenAIConfig is injectable configuration.
type OpenAIConfig struct {
	APIKey string
	// BaseURL overrides the endpoint host (for tests). Optional.
	BaseURL string
	// Model for single-shot transcription. Defaults to "whisper-1".
	Model string
	// StreamModel for TranscribeStream (must support stream=true).
	// Defaults to "gpt-4o-mini-transcribe".
	StreamModel string
	// HTTPClient lets callers inject a custom client.
	HTTPClient *http.Client
}

// NewOpenAI constructs an OpenAI connector.
func NewOpenAI(cfg OpenAIConfig) (*OpenAI, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("stt: OpenAI APIKey is required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultOpenAIBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultOpenAIModel
	}
	streamModel := cfg.StreamModel
	if streamModel == "" {
		streamModel = defaultOpenAIStreamModel
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	return &OpenAI{
		httpClient:  client,
		baseURL:     base,
		apiKey:      cfg.APIKey,
		model:       model,
		streamModel: streamModel,
	}, nil
}

// DefaultOpenAIConnector reads OPENAI_API_KEY, OPENAI_STT_MODEL, and
// OPENAI_STT_STREAM_MODEL from the environment.
func DefaultOpenAIConnector(_ context.Context) (*OpenAI, error) {
	return NewOpenAI(OpenAIConfig{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       os.Getenv("OPENAI_STT_MODEL"),
		StreamModel: os.Getenv("OPENAI_STT_STREAM_MODEL"),
	})
}

// openaiVerboseResponse is the JSON shape returned when we ask for
// response_format=verbose_json (the default for single-shot here).
type openaiVerboseResponse struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	Segments []struct {
		Text  string  `json:"text"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"segments,omitempty"`
}

// filenameFor picks a filename extension for the multipart upload. OpenAI
// uses the filename's extension to sniff the format.
func filenameFor(format audio.AudioFormat) string {
	ext := string(format)
	if ext == "" {
		ext = "wav"
	}
	return "audio." + ext
}

// buildMultipart constructs a multipart/form-data body carrying the audio
// file plus form fields. Returns the body, content-type header value.
func buildMultipart(fields map[string]string, audioData []byte, format audio.AudioFormat) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	// file part — must be named "file" per OpenAI's API.
	filePartHeader := textproto.MIMEHeader{}
	filePartHeader.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename=%q`, filenameFor(format)))
	mime := format.MIMEType()
	if mime == "" {
		mime = "application/octet-stream"
	}
	filePartHeader.Set("Content-Type", mime)

	filePart, err := mw.CreatePart(filePartHeader)
	if err != nil {
		return nil, "", fmt.Errorf("stt: multipart file part: %w", err)
	}
	if _, err := filePart.Write(audioData); err != nil {
		return nil, "", fmt.Errorf("stt: write audio: %w", err)
	}

	// plain form fields
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := mw.WriteField(k, v); err != nil {
			return nil, "", fmt.Errorf("stt: write field %s: %w", k, err)
		}
	}
	if err := mw.Close(); err != nil {
		return nil, "", fmt.Errorf("stt: close multipart: %w", err)
	}
	return body, mw.FormDataContentType(), nil
}

// Transcribe implements STTConnector.
func (o *OpenAI) Transcribe(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResult, error) {
	if req == nil || len(req.Audio) == 0 {
		return nil, errors.New("stt: TranscriptionRequest.Audio is required")
	}
	model := req.Model
	if model == "" {
		model = o.model
	}

	fields := map[string]string{
		"model":           model,
		"language":        req.Language,
		"prompt":          req.Prompt,
		"response_format": "verbose_json",
	}
	if req.Temperature > 0 {
		fields["temperature"] = strconv.FormatFloat(req.Temperature, 'f', -1, 64)
	}

	body, contentType, err := buildMultipart(fields, req.Audio, req.Format)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.baseURL+"/v1/audio/transcriptions", body)
	if err != nil {
		return nil, fmt.Errorf("stt: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("Content-Type", contentType)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		slog.ErrorContext(ctx, "openai transcribe failed", "error", err)
		return nil, fmt.Errorf("stt: http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, readOpenAIError(resp)
	}

	var parsed openaiVerboseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("stt: decode: %w", err)
	}
	result := &TranscriptionResult{
		Text:     parsed.Text,
		Language: parsed.Language,
	}
	if len(parsed.Segments) > 0 {
		result.Segments = make([]TranscriptSegment, len(parsed.Segments))
		for i, s := range parsed.Segments {
			result.Segments[i] = TranscriptSegment{Text: s.Text, Start: s.Start, End: s.End}
		}
	}
	return result, nil
}

// TranscribeStream implements STTConnector. The streaming shape here is:
//
//   - caller pushes audio bytes via Send (we buffer them in-process),
//   - caller signals end-of-audio via CloseSend,
//   - we POST the buffered audio with stream=true,
//   - we consume the SSE response and emit EventDelta/EventFinal.
//
// This is "streaming-out" only. For live bidirectional streaming (live
// mic in + deltas out) you want OpenAI's Realtime API over a WebSocket,
// which is out of scope for this connector today.
func (o *OpenAI) TranscribeStream(ctx context.Context, params *StreamParams) (*TranscriptionStream, error) {
	if params == nil {
		return nil, errors.New("stt: StreamParams is required")
	}
	model := params.Model
	if model == "" {
		model = o.streamModel
	}

	// State shared between producer (Send/CloseSend) and consumer (Next/Event).
	var (
		mu       sync.Mutex
		buf      bytes.Buffer
		sendDone = make(chan struct{}) // closed on CloseSend
		closed   = make(chan struct{}) // closed on Close
	)

	events := make(chan TranscriptEvent, 8)
	var (
		current   TranscriptEvent
		streamErr error
		errMu     sync.Mutex
	)
	setErr := func(err error) {
		errMu.Lock()
		defer errMu.Unlock()
		if streamErr == nil {
			streamErr = err
		}
	}
	getErr := func() error {
		errMu.Lock()
		defer errMu.Unlock()
		return streamErr
	}

	// Consumer-facing: one goroutine waits for sendDone, then POSTs the
	// buffered audio with stream=true, parses SSE, and pushes events.
	go func() {
		defer close(events)
		select {
		case <-sendDone:
		case <-ctx.Done():
			setErr(ctx.Err())
			return
		case <-closed:
			return
		}

		mu.Lock()
		audioBytes := buf.Bytes()
		mu.Unlock()
		if len(audioBytes) == 0 {
			setErr(errors.New("stt: no audio sent before CloseSend"))
			return
		}

		fields := map[string]string{
			"model":    model,
			"language": params.Language,
			"prompt":   params.Prompt,
			"stream":   "true",
		}
		body, contentType, err := buildMultipart(fields, audioBytes, params.Format)
		if err != nil {
			setErr(err)
			return
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			o.baseURL+"/v1/audio/transcriptions", body)
		if err != nil {
			setErr(fmt.Errorf("stt: build stream request: %w", err))
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
		httpReq.Header.Set("Content-Type", contentType)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := o.httpClient.Do(httpReq)
		if err != nil {
			setErr(fmt.Errorf("stt: stream http: %w", err))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			setErr(readOpenAIError(resp))
			return
		}

		if err := consumeSSE(resp.Body, events, closed); err != nil {
			setErr(err)
		}
	}()

	return NewTranscriptionStream(
		func(chunk []byte) error {
			select {
			case <-sendDone:
				return errors.New("stt: Send after CloseSend")
			case <-closed:
				return errors.New("stt: Send on closed stream")
			default:
			}
			mu.Lock()
			defer mu.Unlock()
			_, err := buf.Write(chunk)
			return err
		},
		func() error {
			select {
			case <-sendDone:
				// idempotent
			default:
				close(sendDone)
			}
			return nil
		},
		func() bool {
			ev, ok := <-events
			if !ok {
				return false
			}
			current = ev
			return true
		},
		func() TranscriptEvent { return current },
		getErr,
		func() error {
			select {
			case <-closed:
			default:
				close(closed)
			}
			// Ensure sendDone is closed so the goroutine doesn't hang if
			// the caller closes before CloseSend.
			select {
			case <-sendDone:
			default:
				close(sendDone)
			}
			// Drain events so the producer goroutine can exit.
			for range events {
			}
			return nil
		},
	), nil
}

// consumeSSE reads text/event-stream frames from r and pushes
// TranscriptEvents onto out. Frames that don't parse as the expected
// JSON payloads are ignored (keeps us forward-compatible with new event
// types the API may add).
func consumeSSE(r io.Reader, out chan<- TranscriptEvent, closed <-chan struct{}) error {
	scanner := bufio.NewScanner(r)
	// SSE payloads can be large; raise the buffer ceiling.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var accumulated string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		// Blank line ends a frame; we handle on every "data:" line since
		// each frame has exactly one data: field in OpenAI's format.
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			return nil
		}

		var env struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
			Text  string `json:"text"`
		}
		if err := json.Unmarshal([]byte(payload), &env); err != nil {
			continue // ignore unparseable frames
		}
		switch env.Type {
		case "transcript.text.delta":
			accumulated += env.Delta
			ev := TranscriptEvent{Type: EventDelta, Delta: env.Delta, Text: accumulated}
			select {
			case out <- ev:
			case <-closed:
				return nil
			}
		case "transcript.text.done":
			final := env.Text
			if final == "" {
				final = accumulated
			}
			ev := TranscriptEvent{Type: EventFinal, Text: final}
			select {
			case out <- ev:
			case <-closed:
				return nil
			}
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stt: read sse: %w", err)
	}
	return nil
}

// readOpenAIError parses the non-2xx JSON error envelope.
func readOpenAIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return fmt.Errorf("stt: openai %d: %s", resp.StatusCode, parsed.Error.Message)
	}
	return fmt.Errorf("stt: openai %d: %s", resp.StatusCode, bytes.TrimSpace(body))
}
