package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	audio "github.com/cannonball10/foundation/schemas/audio"
)

func newOpenAIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle("/v1/audio/transcriptions", handler)
	return httptest.NewServer(mux)
}

func newOpenAIClient(t *testing.T, srv *httptest.Server) *OpenAI {
	t.Helper()
	conn, err := NewOpenAI(OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAI: %v", err)
	}
	return conn
}

func TestNewOpenAI_RequiresAPIKey(t *testing.T) {
	if _, err := NewOpenAI(OpenAIConfig{}); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestTranscribe_SendsMultipartAndParsesVerboseJSON(t *testing.T) {
	var gotAuth, gotContentType, gotModel, gotLanguage string
	srv := newOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		}
		gotModel = r.FormValue("model")
		gotLanguage = r.FormValue("language")

		// Make sure the file part is present and non-empty.
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
		}
		if file != nil {
			data, _ := io.ReadAll(file)
			if len(data) == 0 {
				t.Error("file part empty")
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openaiVerboseResponse{
			Text:     "hello world",
			Language: "en",
			Segments: []struct {
				Text  string  `json:"text"`
				Start float64 `json:"start"`
				End   float64 `json:"end"`
			}{{Text: "hello", Start: 0, End: 0.5}, {Text: " world", Start: 0.5, End: 1.0}},
		})
	})
	defer srv.Close()
	conn := newOpenAIClient(t, srv)

	res, err := conn.Transcribe(context.Background(), &TranscriptionRequest{
		Audio:    []byte("RIFFFAKEWAV"),
		Format:   audio.FormatWAV,
		Language: "en",
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data;") {
		t.Errorf("content-type = %q", gotContentType)
	}
	if gotModel != defaultOpenAIModel {
		t.Errorf("model = %q; want %q", gotModel, defaultOpenAIModel)
	}
	if gotLanguage != "en" {
		t.Errorf("language = %q", gotLanguage)
	}
	if res.Text != "hello world" {
		t.Errorf("text = %q", res.Text)
	}
	if res.Language != "en" {
		t.Errorf("lang = %q", res.Language)
	}
	if len(res.Segments) != 2 {
		t.Fatalf("segments len = %d", len(res.Segments))
	}
	if res.Segments[1].End != 1.0 {
		t.Errorf("segment[1].End = %v", res.Segments[1].End)
	}
}

func TestTranscribe_RejectsEmptyAudio(t *testing.T) {
	srv := newOpenAIServer(t, func(http.ResponseWriter, *http.Request) {
		t.Error("server should not be called")
	})
	defer srv.Close()
	conn := newOpenAIClient(t, srv)
	if _, err := conn.Transcribe(context.Background(), &TranscriptionRequest{}); err == nil {
		t.Fatal("expected error for empty audio")
	}
}

func TestTranscribe_PropagatesAPIError(t *testing.T) {
	srv := newOpenAIServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad audio","type":"invalid_request_error"}}`))
	})
	defer srv.Close()
	conn := newOpenAIClient(t, srv)
	_, err := conn.Transcribe(context.Background(), &TranscriptionRequest{
		Audio: []byte("x"), Format: audio.FormatWAV,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad audio") {
		t.Errorf("error = %v", err)
	}
}

func TestTranscribeStream_StreamsDeltasAndFinal(t *testing.T) {
	// SSE payload: two deltas + a done event.
	payload := strings.Join([]string{
		`data: {"type":"transcript.text.delta","delta":"Hel"}`,
		``,
		`data: {"type":"transcript.text.delta","delta":"lo"}`,
		``,
		`data: {"type":"transcript.text.done","text":"Hello"}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	var gotStream, gotModel string
	var gotFileBytes []byte
	srv := newOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		}
		gotStream = r.FormValue("stream")
		gotModel = r.FormValue("model")
		if file, _, err := r.FormFile("file"); err == nil {
			gotFileBytes, _ = io.ReadAll(file)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, line := range strings.Split(payload, "\n") {
			fmt.Fprintln(w, line)
		}
		if flusher != nil {
			flusher.Flush()
		}
	})
	defer srv.Close()
	conn := newOpenAIClient(t, srv)

	stream, err := conn.TranscribeStream(context.Background(), &StreamParams{
		Format: audio.FormatPCM16, SampleRate: 16000,
	})
	if err != nil {
		t.Fatalf("TranscribeStream: %v", err)
	}
	defer stream.Close()

	// Producer: two chunks then CloseSend.
	if err := stream.Send([]byte("chunk1")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := stream.Send([]byte("chunk2")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}

	var deltas []string
	var final string
	for stream.Next() {
		ev := stream.Event()
		switch ev.Type {
		case EventDelta:
			deltas = append(deltas, ev.Delta)
		case EventFinal:
			final = ev.Text
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err: %v", err)
	}

	if gotStream != "true" {
		t.Errorf("stream form field = %q", gotStream)
	}
	if gotModel != defaultOpenAIStreamModel {
		t.Errorf("model = %q; want %q", gotModel, defaultOpenAIStreamModel)
	}
	if string(gotFileBytes) != "chunk1chunk2" {
		t.Errorf("accumulated audio = %q", gotFileBytes)
	}
	if got, want := strings.Join(deltas, ""), "Hello"; got != want {
		t.Errorf("deltas joined = %q; want %q", got, want)
	}
	if final != "Hello" {
		t.Errorf("final = %q", final)
	}
}

func TestTranscribeStream_ErrorsWhenNoAudioSent(t *testing.T) {
	srv := newOpenAIServer(t, func(http.ResponseWriter, *http.Request) {
		t.Error("server should not be called")
	})
	defer srv.Close()
	conn := newOpenAIClient(t, srv)

	stream, err := conn.TranscribeStream(context.Background(), &StreamParams{Format: audio.FormatWAV})
	if err != nil {
		t.Fatalf("TranscribeStream: %v", err)
	}
	defer stream.Close()
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}
	for stream.Next() {
		t.Error("should not yield events")
	}
	if stream.Err() == nil {
		t.Fatal("expected error when no audio was sent")
	}
}
