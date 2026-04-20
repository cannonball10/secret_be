package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	audio "github.com/cannonball10/foundation/schemas/audio"
)

// newTestServer returns an httptest.Server that behaves like the ElevenLabs
// TTS API for the endpoints we actually call. Handler observes what we sent
// and writes fake audio back.
func newTestServer(t *testing.T, onReq func(*http.Request, []byte)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/text-to-speech/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if onReq != nil {
			onReq(r, body)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		if strings.HasSuffix(r.URL.Path, "/stream") {
			// Simulate streaming: write three slices, flushing between.
			flusher, _ := w.(http.Flusher)
			chunks := []string{"aaa", "bbb", "ccc"}
			for _, c := range chunks {
				_, _ = w.Write([]byte(c))
				if flusher != nil {
					flusher.Flush()
				}
			}
			return
		}
		_, _ = w.Write([]byte("FAKE_MP3_BYTES"))
	})
	return httptest.NewServer(mux)
}

func newTestConnector(t *testing.T, srv *httptest.Server) *ElevenLabs {
	t.Helper()
	conn, err := NewElevenLabs(ElevenLabsConfig{
		APIKey:         "test-key",
		BaseURL:        srv.URL,
		DefaultVoiceID: "voice_abc",
	})
	if err != nil {
		t.Fatalf("NewElevenLabs: %v", err)
	}
	return conn
}

func TestNewElevenLabs_RequiresAPIKey(t *testing.T) {
	if _, err := NewElevenLabs(ElevenLabsConfig{DefaultVoiceID: "v"}); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNewElevenLabs_RequiresVoiceID(t *testing.T) {
	if _, err := NewElevenLabs(ElevenLabsConfig{APIKey: "k"}); err == nil {
		t.Fatal("expected error for missing voice ID")
	}
}

func TestSynthesize_HitsCorrectPathAndReturnsAudio(t *testing.T) {
	var gotPath, gotAPIKey, gotQuery string
	var gotBody []byte
	srv := newTestServer(t, func(r *http.Request, body []byte) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAPIKey = r.Header.Get("xi-api-key")
		gotBody = body
	})
	defer srv.Close()

	conn := newTestConnector(t, srv)
	res, err := conn.Synthesize(context.Background(), &SynthesisRequest{
		Text:       "hello world",
		Format:     audio.FormatPCM16,
		SampleRate: audio.SampleRate24kHz,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if got, want := gotPath, "/v1/text-to-speech/voice_abc"; got != want {
		t.Errorf("path = %q; want %q", got, want)
	}
	if gotAPIKey != "test-key" {
		t.Errorf("api key header = %q", gotAPIKey)
	}
	if gotQuery != "output_format=pcm_24000" {
		t.Errorf("query = %q", gotQuery)
	}
	var parsed elevenBody
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if parsed.Text != "hello world" {
		t.Errorf("text = %q", parsed.Text)
	}
	if parsed.ModelID != defaultElevenLabsModel {
		t.Errorf("model = %q; want %q", parsed.ModelID, defaultElevenLabsModel)
	}
	if parsed.VoiceSettings != nil {
		t.Errorf("voice_settings should be omitted when all knobs zero: %+v", parsed.VoiceSettings)
	}
	if string(res.Audio) != "FAKE_MP3_BYTES" {
		t.Errorf("audio = %q", res.Audio)
	}
	if res.Format != audio.FormatPCM16 {
		t.Errorf("format = %q", res.Format)
	}
}

func TestSynthesize_IncludesVoiceSettingsWhenSet(t *testing.T) {
	var gotBody []byte
	srv := newTestServer(t, func(_ *http.Request, body []byte) {
		gotBody = body
	})
	defer srv.Close()
	conn := newTestConnector(t, srv)

	_, err := conn.Synthesize(context.Background(), &SynthesisRequest{
		Text:       "hi",
		Stability:  0.4,
		Similarity: 0.7,
		Style:      0.1,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	var parsed elevenBody
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.VoiceSettings == nil {
		t.Fatal("voice_settings omitted despite non-zero inputs")
	}
	if parsed.VoiceSettings.Stability != 0.4 || parsed.VoiceSettings.SimilarityBoost != 0.7 || parsed.VoiceSettings.Style != 0.1 {
		t.Errorf("voice_settings = %+v", parsed.VoiceSettings)
	}
}

func TestSynthesizeStream_IteratesChunks(t *testing.T) {
	srv := newTestServer(t, nil)
	defer srv.Close()
	conn := newTestConnector(t, srv)

	stream, err := conn.SynthesizeStream(context.Background(), &SynthesisRequest{Text: "stream me"})
	if err != nil {
		t.Fatalf("SynthesizeStream: %v", err)
	}
	defer stream.Close()

	var got []byte
	for stream.Next() {
		got = append(got, stream.Chunk().Data...)
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err: %v", err)
	}
	if string(got) != "aaabbbccc" {
		t.Errorf("assembled = %q", got)
	}
}

func TestSynthesize_PropagatesHTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/text-to-speech/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":{"status":"invalid_api_key"}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	conn := newTestConnector(t, srv)

	_, err := conn.Synthesize(context.Background(), &SynthesisRequest{Text: "x"})
	if err == nil {
		t.Fatal("expected error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention 401: %v", err)
	}
}

func TestSynthesize_RejectsEmptyText(t *testing.T) {
	srv := newTestServer(t, nil)
	defer srv.Close()
	conn := newTestConnector(t, srv)
	if _, err := conn.Synthesize(context.Background(), &SynthesisRequest{Text: ""}); err == nil {
		t.Fatal("expected error for empty text")
	}
}
