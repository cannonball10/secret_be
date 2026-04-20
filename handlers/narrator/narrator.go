// Package narrator turns gameplay cues into a spoken broadcast.
//
// Flow per Speak(cue):
//   1. Render the cue's prompt with game-state variables.
//   2. Call the configured text-inference connector (Claude via the
//      foundation's Anthropic connector) with a Committee-voice
//      system prompt + few-shot examples.
//   3. Trim the script and synthesize it with the configured TTS
//      connector (ElevenLabs).
//   4. Return the script + audio bytes; the caller decides how to
//      broadcast them (see api/narrator_handlers.go).
//
// The service is intentionally stateless. It borrows the game engine
// handle only to look up player display names for cue variables. The
// HTTP handler passes the in-memory audio through a separate
// get-by-id route rather than embedding it — ElevenLabs MP3 for a
// short line is typically 20-50KB, which is fine to keep in-process
// for a few minutes.

package narrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cannonball10/foundation/connectors/inference"
	"github.com/cannonball10/foundation/connectors/tts"
	audioschema "github.com/cannonball10/foundation/schemas/audio"
	inferschema "github.com/cannonball10/foundation/schemas/inference"
)

// CueKind enumerates the script categories the narrator can render.
type CueKind string

const (
	CueOpening          CueKind = "opening"
	CueElectionPassed   CueKind = "election_passed"
	CueElectionRejected CueKind = "election_rejected"
	CueExecution        CueKind = "execution"
	CueClosing          CueKind = "closing"
	// CueCableLeak announces a leaked Cable Phase cable. Vars:
	//   · "author"  — speaker's display name
	//   · "body"    — the verbatim cable text
	CueCableLeak CueKind = "cable_leak"
	// CueCableSilence is what the Committee says when no cable from
	// the round meets the subversion bar. Takes no vars.
	CueCableSilence CueKind = "cable_silence"
	// CueCustom lets a host type raw text; we skip the LLM and feed
	// the text straight to TTS so the demo can showcase the pipeline
	// without burning an inference call.
	CueCustom CueKind = "custom"
)

// Cue is the input to Speak. Vars populate the cue prompt template
// (e.g. {"president": "Jordan"}). For CueCustom, set Vars["text"].
type Cue struct {
	Kind CueKind
	Vars map[string]string
}

// Result is the output of Speak. Audio is the encoded MP3 bytes;
// CueID is a stable identifier the caller can use to cache + serve it.
type Result struct {
	CueID  string
	Script string
	Audio  []byte
	Format audioschema.AudioFormat
	TookMS int64
}

// Narrator wires an inference connector to a TTS connector and
// exposes a single Speak method the HTTP layer invokes.
type Narrator struct {
	infer    inference.TextInferenceConnector
	tts      tts.TTSConnector
	model    string
	voiceID  string
	ttsModel string

	seq uint64
	mu  sync.Mutex
}

// Config bundles the optional knobs. Zero values use sensible defaults.
type Config struct {
	// Model is the Claude model id (e.g. "claude-sonnet-4-6"). Zero
	// falls back to "claude-sonnet-4-6".
	Model string
	// VoiceID is the ElevenLabs voice UUID. Zero = use the TTS
	// connector's configured default (reads ELEVENLABS_VOICE_ID at
	// construction time).
	VoiceID string
	// TTSModel is the ElevenLabs synthesis model (e.g.
	// "eleven_turbo_v2_5"). Zero = provider default.
	TTSModel string
}

// New builds a Narrator with the given connectors and config.
func New(infer inference.TextInferenceConnector, ttsConn tts.TTSConnector, cfg Config) *Narrator {
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &Narrator{
		infer:    infer,
		tts:      ttsConn,
		model:    model,
		voiceID:  cfg.VoiceID,
		ttsModel: cfg.TTSModel,
	}
}

// Speak generates a script for the cue, synthesises it, and returns
// the result. On any provider failure the error is wrapped with the
// stage (inference vs tts) so callers can fall back cleanly — e.g.
// the host can play an HTML-TTS placeholder if ElevenLabs is down.
func (n *Narrator) Speak(ctx context.Context, cue Cue) (*Result, error) {
	started := time.Now()

	// Render script text. CueCustom skips the LLM — the host typed
	// exactly what they want spoken.
	var script string
	if cue.Kind == CueCustom {
		script = strings.TrimSpace(firstNonEmpty(cue.Vars["text"], ""))
		if script == "" {
			return nil, fmt.Errorf("narrator: custom cue has no text")
		}
	} else {
		userPrompt := promptForCue(cue.Kind, cue.Vars)
		req := &inferschema.CompletionRequest{
			Model: n.model,
			Messages: []inferschema.Message{
				{Role: inferschema.MessageRole_System, Content: systemPrompt},
				{Role: inferschema.MessageRole_User, Content: userPrompt},
			},
			MaxTokens:   250,
			Temperature: floatPtr(0.7),
		}
		resp, err := n.infer.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("narrator: inference: %w", err)
		}
		script = strings.TrimSpace(resp.Message.TextContent())
		script = sanitize(script)
	}

	if script == "" {
		return nil, fmt.Errorf("narrator: empty script")
	}

	// Synthesize to MP3. ElevenLabs default is MP3; we leave Format
	// unset and let the connector choose.
	syn, err := n.tts.Synthesize(ctx, &tts.SynthesisRequest{
		Text:    script,
		VoiceID: n.voiceID,
		Model:   n.ttsModel,
	})
	if err != nil {
		return nil, fmt.Errorf("narrator: tts: %w", err)
	}

	return &Result{
		CueID:  n.nextID(),
		Script: script,
		Audio:  syn.Audio,
		Format: syn.Format,
		TookMS: time.Since(started).Milliseconds(),
	}, nil
}

// Cable is one intercepted message the ranker should score. The engine
// collects these by querying ChatMessage rows for the current round.
type Cable struct {
	MessageID string
	Author    string
	Body      string
}

// ScoredCable is the ranker's output — one entry per input, with a
// "subversion" score in [0, 10] where 10 means "smoking gun" and 0
// means innocuous small talk. Reason is the ranker's brief
// justification; the engine ignores it at runtime but it's useful for
// debugging and for a future "Committee's notes" UI.
type ScoredCable struct {
	MessageID string  `json:"id"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason,omitempty"`
}

// RankCables asks the LLM to score each cable on subversiveness. The
// returned slice preserves input order and always has len == len(cables)
// — entries the model failed to score default to Score = 0 with an
// empty reason so the caller can still pick a top without special
// handling for partial output.
//
// On inference failure RankCables returns the error; callers should
// fall back to either a deterministic rule (longest cable) or the
// "silence" path when ranking is unavailable.
func (n *Narrator) RankCables(ctx context.Context, cables []Cable) ([]ScoredCable, error) {
	if len(cables) == 0 {
		return nil, nil
	}
	prompt := buildRankPrompt(cables)
	req := &inferschema.CompletionRequest{
		Model: n.model,
		Messages: []inferschema.Message{
			{Role: inferschema.MessageRole_System, Content: rankerSystemPrompt},
			{Role: inferschema.MessageRole_User, Content: prompt},
		},
		MaxTokens:   600,
		Temperature: floatPtr(0.2),
	}
	resp, err := n.infer.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("narrator: rank inference: %w", err)
	}
	raw := strings.TrimSpace(resp.Message.TextContent())
	scored, err := parseScoredCables(raw)
	if err != nil {
		return nil, fmt.Errorf("narrator: rank parse: %w (raw=%q)", err, raw)
	}
	// Align output with input order. Missing ids → score 0.
	byID := make(map[string]ScoredCable, len(scored))
	for _, s := range scored {
		byID[s.MessageID] = s
	}
	out := make([]ScoredCable, len(cables))
	for i, c := range cables {
		if s, ok := byID[c.MessageID]; ok {
			out[i] = s
			// Guard against out-of-range scores from the model.
			if out[i].Score < 0 {
				out[i].Score = 0
			}
			if out[i].Score > 10 {
				out[i].Score = 10
			}
			continue
		}
		out[i] = ScoredCable{MessageID: c.MessageID}
	}
	return out, nil
}

func (n *Narrator) nextID() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.seq++
	return fmt.Sprintf("cue-%d-%d", time.Now().UnixNano(), n.seq)
}

// sanitize strips stage directions + stray exclamation marks. Claude
// usually obeys the "no exclamation marks" rule but belt-and-suspenders
// is cheap insurance — one stray "!" ruins the register.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "!", ".")
	// Drop wrapping quotes the model sometimes adds despite instruction.
	s = strings.Trim(s, ` "'`)
	// Collapse whitespace.
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func floatPtr(f float64) *float64 { return &f }

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
