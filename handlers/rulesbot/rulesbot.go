// Package rulesbot answers natural-language questions about Replicant
// rules from a player's mobile device. It's a thin wrapper around the
// shared TextInferenceConnector (same Anthropic client the narrator
// uses) with a dedicated system prompt that teaches the model the
// rules and — critically — forbids it from revealing other players'
// secret roles.
//
// The handler accepts a per-call PlayerContext so the bot can answer
// "what does my role do?" accurately for the asking player, but the
// system prompt explicitly instructs it to refuse questions about
// anyone else's role.

package rulesbot

import (
	"context"
	"fmt"
	"strings"

	"github.com/cannonball10/foundation/connectors/inference"
	inferschema "github.com/cannonball10/foundation/schemas/inference"
)

// Bot wires an inference connector to the rules-Q&A system prompt.
type Bot struct {
	infer inference.TextInferenceConnector
	model string
}

// Config bundles the optional knobs.
type Config struct {
	// Model is the Claude model id. Zero falls back to a small/fast
	// default — answers are short text, no tool use.
	Model string
}

// New constructs a rules bot. Caller passes the same inference
// connector used by the narrator.
func New(infer inference.TextInferenceConnector, cfg Config) *Bot {
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &Bot{infer: infer, model: model}
}

// PlayerContext is the asker's own private info. Only their role +
// country are shared with the LLM. The bot is instructed to never
// speculate about anyone else.
type PlayerContext struct {
	DisplayName string
	Country     string
	// Role is the ASKER's role, one of "human", "ai", "rogue",
	// "singularity", or empty before role assignment.
	Role string
	// EnableSingularity tells the bot whether the 3-faction variant
	// is active so its answers don't mention the Singularity role
	// in 2-faction games.
	EnableSingularity bool
	// CablePhaseOn tells the bot whether the Cable Phase mechanic
	// runs every round so it doesn't coach players on a phase that
	// won't happen.
	CablePhaseOn bool
}

// Ask returns the LLM's answer to a natural-language question. Empty
// or overly long questions are rejected up-front so a stray tap
// doesn't burn an inference call.
func (b *Bot) Ask(ctx context.Context, question string, who PlayerContext) (string, error) {
	q := strings.TrimSpace(question)
	if q == "" {
		return "", fmt.Errorf("rulesbot: empty question")
	}
	if len(q) > 600 {
		q = q[:600]
	}
	system := buildSystemPrompt(who)
	req := &inferschema.CompletionRequest{
		Model: b.model,
		Messages: []inferschema.Message{
			{Role: inferschema.MessageRole_System, Content: system},
			{Role: inferschema.MessageRole_User, Content: q},
		},
		MaxTokens:   500,
		Temperature: floatPtr(0.3),
	}
	resp, err := b.infer.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("rulesbot: %w", err)
	}
	return strings.TrimSpace(resp.Message.TextContent()), nil
}

func floatPtr(f float64) *float64 { return &f }

// FAQ returns the canned starter prompts the mobile UI shows above
// the compose input. Returning them from the server means the bot's
// behaviour and its surfaced FAQ stay in sync on a single code path.
func FAQ(who PlayerContext) []string {
	base := []string{
		"What are the factions and how do I win?",
		"How does the election work? What's Yea vs Nay?",
		"What do the presidential powers do?",
		"What does my role do?",
	}
	if who.CablePhaseOn {
		base = append(base, "What is Cable Phase and how do leaks work?")
	}
	return base
}
