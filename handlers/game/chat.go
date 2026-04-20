package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// ChatChannel names a conversation scope. "ai" is the replicant cabal
// whisper room; additional channels (e.g. "singularity", "all-day")
// can be added without changing the delivery semantics — each new
// channel just extends the audience check in SendChat.
type ChatChannel string

const (
	// ChannelAI is the private whisper channel shared by all AI-faction
	// roles (replicants + the rogue/prime). Humans never see it.
	ChannelAI ChatChannel = "ai"

	// ChannelCable is a write-only submission channel used during the
	// Cable Phase. The server persists the message with the current
	// GovernmentID so the phase-end ranker can pick one to leak, then
	// whispers an ack back to the sender only. No other player sees
	// the content at send time — the whole point of the channel is
	// private typing under cover of the other players' typing.
	ChannelCable ChatChannel = "cable"
)

// ChatMessagePayload is emitted as a whisper per eligible recipient,
// keeping the fan-out inside the existing audience model rather than
// adding a faction-scoped audience to the Hub.
type ChatMessagePayload struct {
	MessageID         string      `json:"messageId"`
	Channel           ChatChannel `json:"channel"`
	AuthorPlayerID    string      `json:"authorPlayerId"`
	AuthorDisplayName string      `json:"authorDisplayName"`
	Body              string      `json:"body"`
	SentAt            string      `json:"sentAt"` // RFC3339
	// GovernmentID is populated for ChannelCable submissions so the
	// host UI can show a per-round counter. Empty for channels that
	// aren't round-scoped (cabal).
	GovernmentID string `json:"governmentId,omitempty"`
	// Ack is true when this payload is a write-ack sent back to the
	// author of a ChannelCable submission. Other receivers never see
	// Ack=true envelopes.
	Ack bool `json:"ack,omitempty"`
}

// ErrChannelAccessDenied is returned when a player tries to post to a
// channel they aren't a member of.
var ErrChannelAccessDenied = fmt.Errorf("%w: channel access denied", ErrIneligibleCandidate)

// SendChat posts a message to the given channel. The sender must be
// in the game; for channel-restricted rooms (e.g. "ai") the sender
// must also be a member of that faction.
//
// Each eligible recipient gets an individual whisper envelope. This
// is slightly chattier than a single faction-scoped broadcast, but
// keeps the Hub's delivery model (broadcast / per-player whisper)
// unchanged — a win for the multi-node Redis rewrite later.
func (h *GameHandler) SendChat(ctx context.Context, gameID, senderPlayerID string, channel ChatChannel, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("%w: empty chat body", ErrInvalidTransition)
	}
	if len(body) > 500 {
		body = body[:500]
	}

	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if game.Status == secrethitler.GameStatusLobby || game.Status == secrethitler.GameStatusCompleted {
		return fmt.Errorf("%w: chat unavailable outside active play", ErrInvalidTransition)
	}

	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	sender := findPlayerByID(players, senderPlayerID)
	if sender == nil {
		return ErrPlayerNotFound
	}
	if !sender.IsAlive {
		return fmt.Errorf("%w: dead players may not send", ErrChannelAccessDenied)
	}

	// ChannelCable is a private submission — gated to Cable Phase, no
	// broadcast fan-out at send time (the phase-end ranker in step 3c
	// decides whether to leak one). Persist the message so the ranker
	// has something to score, then whisper an ack to the sender so
	// their compose UI can clear.
	if channel == ChannelCable {
		if game.Phase != secrethitler.PhaseCablePhase {
			return fmt.Errorf("%w: cables only during Cable Phase", ErrInvalidTransition)
		}
		msg := models.NewChatMessage(gameID, string(channel), sender.PlayerID, sender.DisplayName, body)
		msg.GovernmentID = game.CurrentGovernmentID
		msg.SubmittedAt = h.clock.Now().UTC().Format(time.RFC3339)
		if err := h.db.Upsert(ctx, nil, msg); err != nil {
			return err
		}
		ack := ChatMessagePayload{
			MessageID:         msg.MessageID,
			Channel:           channel,
			AuthorPlayerID:    sender.PlayerID,
			AuthorDisplayName: sender.DisplayName,
			Body:              body,
			SentAt:            msg.SubmittedAt,
			GovernmentID:      msg.GovernmentID,
			Ack:               true,
		}
		ev := models.NewGameEvent(gameID, secrethitler.EventChatMessage, sender.PlayerID).
			WithTarget(sender.PlayerID)
		h.whisper(ctx, ev, sender.PlayerID, ack)
		return nil
	}

	// Non-cable channels (cabal today): whisper to every reader of the
	// channel. Persistence lets the post-game passport show the full
	// cabal conversation log.
	audience := channelAudience(channel, players)
	if !containsPlayer(audience, sender.PlayerID) {
		return ErrChannelAccessDenied
	}

	now := h.clock.Now().UTC().Format(time.RFC3339)
	msg := models.NewChatMessage(gameID, string(channel), sender.PlayerID, sender.DisplayName, body)
	msg.SubmittedAt = now
	if err := h.db.Upsert(ctx, nil, msg); err != nil {
		return err
	}
	payload := ChatMessagePayload{
		MessageID:         msg.MessageID,
		Channel:           channel,
		AuthorPlayerID:    sender.PlayerID,
		AuthorDisplayName: sender.DisplayName,
		Body:              body,
		SentAt:            now,
	}
	for _, p := range audience {
		ev := models.NewGameEvent(gameID, secrethitler.EventChatMessage, sender.PlayerID).
			WithTarget(p.PlayerID)
		h.whisper(ctx, ev, p.PlayerID, payload)
	}
	return nil
}

// channelAudience returns the set of players who can read a channel.
// For the AI cabal channel: every alive player whose role is AI or
// Rogue. Dead members lose access — no from-beyond-the-grave chat.
func channelAudience(ch ChatChannel, players []*models.Player) []*models.Player {
	out := make([]*models.Player, 0, len(players))
	switch ch {
	case ChannelAI:
		for _, p := range players {
			if !p.IsAlive {
				continue
			}
			if p.Role == secrethitler.RoleAI || p.Role == secrethitler.RoleRogue {
				out = append(out, p)
			}
		}
	}
	return out
}

func containsPlayer(players []*models.Player, id string) bool {
	for _, p := range players {
		if p.PlayerID == id {
			return true
		}
	}
	return false
}

