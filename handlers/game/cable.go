// Package game — cable.go is the Cable Phase handler.
//
// The Cable Phase is a short interstitial between chancellor nomination
// and the election vote. While it's open, players can submit private
// cables (the LLM scorer picks one to leak at phase-end). This file
// owns phase ingress/egress and the post-phase leak flow.
package game

import (
	"context"
	"encoding/base64"
	"log/slog"

	"github.com/cannonball10/foundation/handlers/narrator"
	"github.com/cannonball10/foundation/models"
	audioschema "github.com/cannonball10/foundation/schemas/audio"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// advanceFromCablePhase closes the current Cable Phase, runs the
// leak flow (if a narrator is configured), and moves the game to
// election. Called from progress.advance() when the cable deadline
// expires or the host forces progression.
//
// Failure modes, in decreasing severity:
//   - DB query for cables fails: log + close silently, keep playing.
//   - Narrator ranking fails: fall back to longest cable (deterministic).
//   - Narrator speech synthesis fails: still emit cable_leaked with
//     text, skip audio. The host UI renders a text-only leak.
//
// The phase-transition itself is never blocked by narrator errors —
// players expect to vote after cable phase regardless.
func (h *GameHandler) advanceFromCablePhase(ctx context.Context, game *models.Game, reason ProgressReason) error {
	if game.Phase != secrethitler.PhaseCablePhase {
		return ErrInvalidTransition
	}

	leak := h.runCableLeak(ctx, game)

	closedEv := models.NewGameEvent(game.GameID, secrethitler.EventCablePhaseClosed, "")
	h.broadcast(ctx, closedEv, CablePhaseClosedPayload{
		GovernmentID:     game.CurrentGovernmentID,
		Reason:           reason,
		LeakedMessageID:  leak.leakedMessageID,
		TotalSubmissions: leak.totalSubmissions,
	})

	// Fire the leak envelopes after cable_phase_closed so clients
	// know the phase is over before the narrator starts speaking.
	if leak.payload != nil {
		leakEv := models.NewGameEvent(game.GameID, secrethitler.EventCableLeaked, "")
		h.broadcast(ctx, leakEv, *leak.payload)
	}
	if leak.narratorPayload != nil {
		nEv := models.NewGameEvent(game.GameID, secrethitler.EventNarratorSpeak, "")
		h.broadcast(ctx, nEv, *leak.narratorPayload)
	}

	h.setPhase(ctx, game, secrethitler.PhaseElection, reason)
	return h.saveGame(ctx, game)
}

// cableLeakResult bundles every output the cable leak flow produces,
// so advanceFromCablePhase can fire events in the right order while
// keeping the business logic in one function.
type cableLeakResult struct {
	totalSubmissions int
	leakedMessageID  string
	payload          *CableLeakedPayload
	narratorPayload  *NarratorSpeakPayload
}

// runCableLeak is the per-phase business logic: fetch cables, rank
// them, apply the silence roll, broadcast the result via the narrator.
// Returns a result bundle with nil fields for the "no leak happened"
// path so the caller's event fan-out is straightforward.
func (h *GameHandler) runCableLeak(ctx context.Context, game *models.Game) cableLeakResult {
	result := cableLeakResult{}

	if h.narrator == nil || game.CurrentGovernmentID == "" {
		// No narrator configured, or no government — nothing to leak.
		return result
	}

	cables, err := h.loadCablesForGovernment(ctx, game.GameID, game.CurrentGovernmentID)
	if err != nil {
		slog.Warn("cable: loadCables failed", "err", err, "gameId", game.GameID)
		return result
	}
	result.totalSubmissions = len(cables)
	if len(cables) == 0 {
		return result
	}

	// Silence roll: if the Committee decided to go silent, emit a
	// silence narrator script but no cable_leaked payload with a
	// quoted body.
	silenced := h.rng.Float64() < game.Rules.CableLeakSilenceChance
	if silenced {
		result.payload = &CableLeakedPayload{
			GovernmentID: game.CurrentGovernmentID,
			Silenced:     true,
		}
		if np := h.synthesiseLeak(ctx, narrator.Cue{Kind: narrator.CueCableSilence}); np != nil {
			result.narratorPayload = np
		}
		return result
	}

	// Rank the cables. On ranker failure, fall back to the longest
	// cable — a crude but deterministic stand-in that keeps the phase
	// moving when the LLM is down.
	rankerInput := make([]narrator.Cable, len(cables))
	for i, c := range cables {
		rankerInput[i] = narrator.Cable{MessageID: c.MessageID, Author: c.AuthorDisplayName, Body: c.Body}
	}
	scored, err := h.narrator.RankCables(ctx, rankerInput)
	var topID string
	var topScore float64
	if err != nil {
		slog.Warn("cable: RankCables failed, falling back to longest", "err", err)
		topID = pickLongestCable(cables)
	} else {
		topID, topScore = pickTopScored(scored)
	}
	top := findCableByID(cables, topID)
	if top == nil {
		// Shouldn't happen — ranker returned an id we didn't submit.
		slog.Warn("cable: pickTop returned unknown id", "id", topID)
		return result
	}

	// Persist the Leaked flag + score so the post-game passport can
	// surface which cables the Committee flagged.
	top.Leaked = true
	top.SubversionScore = topScore
	if err := h.db.Upsert(ctx, nil, top); err != nil {
		slog.Warn("cable: persist leaked flag failed", "err", err)
	}

	result.leakedMessageID = top.MessageID
	result.payload = &CableLeakedPayload{
		GovernmentID:    game.CurrentGovernmentID,
		Silenced:        false,
		MessageID:       top.MessageID,
		AuthorPlayerID:  top.AuthorPlayerID,
		Author:          top.AuthorDisplayName,
		Body:            top.Body,
		SubversionScore: topScore,
	}
	if np := h.synthesiseLeak(ctx, narrator.Cue{
		Kind: narrator.CueCableLeak,
		Vars: map[string]string{"author": top.AuthorDisplayName, "body": top.Body},
	}); np != nil {
		result.narratorPayload = np
	}
	return result
}

// synthesiseLeak wraps narrator.Speak, returning nil on failure so
// the caller can just skip the audio envelope. Audio failures never
// block phase progression.
func (h *GameHandler) synthesiseLeak(ctx context.Context, cue narrator.Cue) *NarratorSpeakPayload {
	r, err := h.narrator.Speak(ctx, cue)
	if err != nil {
		slog.Warn("cable: narrator.Speak failed", "err", err, "cue", cue.Kind)
		return nil
	}
	return &NarratorSpeakPayload{
		CueID:    r.CueID,
		Cue:      string(cue.Kind),
		Script:   r.Script,
		AudioURL: audioDataURL(r.Audio, r.Format),
		AudioID:  r.CueID,
		Format:   string(r.Format),
		TookMS:   r.TookMS,
	}
}

// audioDataURL builds a data: URL from the synthesised audio bytes.
// Mirrors the format the /host/narrate endpoint uses so clients can
// play both paths through the same decoder.
func audioDataURL(audio []byte, format audioschema.AudioFormat) string {
	mime := format.MIMEType()
	if mime == "" {
		mime = "audio/mpeg"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(audio)
}

// loadCablesForGovernment queries every ChatMessage row for the given
// government id. Uses the MSG#{govId}# prefix in the sort key so the
// cost is O(|cables for this phase|), not a full scan.
func (h *GameHandler) loadCablesForGovernment(ctx context.Context, gameID, govID string) ([]*models.ChatMessage, error) {
	prefix := "MSG#" + govID + "#"
	out, err := h.db.Query(ctx, nil, database.QueryInput{
		PartitionKey: models.ChatMessageKeys.PK(gameID),
		SortKey:      &database.SortKeyCondition{BeginsWith: &prefix},
	}, database.QueryOptions{})
	if err != nil {
		return nil, err
	}
	cables := make([]*models.ChatMessage, 0, len(out.Models))
	for _, m := range out.Models {
		if msg, ok := m.(*models.ChatMessage); ok && msg.Channel == string(ChannelCable) {
			cables = append(cables, msg)
		}
	}
	return cables, nil
}

// pickTopScored returns the id and score of the highest-scored cable.
// Ties break toward earlier entries (which, thanks to ULID sort keys,
// correspond to earlier-submitted cables).
func pickTopScored(scored []narrator.ScoredCable) (string, float64) {
	var topID string
	var top float64 = -1
	for _, s := range scored {
		if s.Score > top {
			top = s.Score
			topID = s.MessageID
		}
	}
	if top < 0 {
		return "", 0
	}
	return topID, top
}

// pickLongestCable is the fallback when ranking fails. Longer cables
// are correlated with subversiveness in practice — a terse "aye"
// reveals nothing, while a five-sentence plea to flip a vote reveals
// a lot. Cheap heuristic, deterministic, keeps the phase moving.
func pickLongestCable(cables []*models.ChatMessage) string {
	var longestID string
	longest := -1
	for _, c := range cables {
		if len(c.Body) > longest {
			longest = len(c.Body)
			longestID = c.MessageID
		}
	}
	return longestID
}

func findCableByID(cables []*models.ChatMessage, id string) *models.ChatMessage {
	for _, c := range cables {
		if c.MessageID == id {
			return c
		}
	}
	return nil
}
