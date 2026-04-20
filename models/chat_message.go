package models

import (
	"github.com/cannonball10/foundation/utils"
)

// ChatMessageKeys provides key construction for the ChatMessage model.
// PK: GAME#{gameId}, SK: MSG#{governmentId}#{ulid}
//
// The governmentId prefix in the sort key lets us Query all messages
// from a single Cable Phase in one call (begins_with "MSG#{govId}#")
// without scanning the entire event log. ULID keeps ordering stable.
var ChatMessageKeys = NewKeyBuilder("GAME#", "MSG#")

// ChatMessage is a persisted chat payload. Cabal messages, DMs, and
// Cable Phase submissions all serialise through this one type — the
// Channel field discriminates.
//
// Only Cable Phase submissions need persistence beyond SSE fan-out
// (the LLM scorer reads from DynamoDB at phase-end to pick a leak),
// but storing all channels gives us a full conversation log for the
// post-game passport + makes moderation tooling straightforward.
type ChatMessage struct {
	Timestamps

	MessageID   string `json:"messageId"`
	GameID      string `json:"gameId"`
	Channel     string `json:"channel"`
	AuthorPlayerID    string `json:"authorPlayerId"`
	AuthorDisplayName string `json:"authorDisplayName"`
	// GovernmentID associates a Cable Phase submission with the
	// nomination it's attached to, so the phase-end scorer can query
	// only this round's cables. Empty for always-on channels (cabal).
	GovernmentID string `json:"governmentId,omitempty"`
	// RecipientPlayerID is set for DM channels, empty otherwise.
	RecipientPlayerID string `json:"recipientPlayerId,omitempty"`
	Body              string `json:"body"`
	// SubmittedAt is the original wall-clock timestamp. Timestamps
	// already tracks CreatedAt/UpdatedAt for the Dynamo record;
	// SubmittedAt is the game-clock value (via h.clock) that the
	// engine injects so tests with FakeClock stay deterministic.
	SubmittedAt string `json:"submittedAt"`
	// Leaked flags the one cable the Committee broadcast at
	// phase-end. Set by the LLM-ranker path in step 3c.
	Leaked bool `json:"leaked,omitempty"`
	// SubversionScore is the LLM's rating (0-10) of how provocative
	// the message is. Set at phase-end by the ranker.
	SubversionScore float64 `json:"subversionScore,omitempty"`
}

// NewChatMessage creates a ChatMessage with an auto-generated ULID.
func NewChatMessage(gameID, channel, authorPlayerID, authorDisplayName, body string) *ChatMessage {
	return &ChatMessage{
		Timestamps:        NewTimestamps(),
		MessageID:         utils.GenerateULID(),
		GameID:            gameID,
		Channel:           channel,
		AuthorPlayerID:    authorPlayerID,
		AuthorDisplayName: authorDisplayName,
		Body:              body,
	}
}

func (m *ChatMessage) PK() string { return ChatMessageKeys.PK(m.GameID) }
func (m *ChatMessage) SK() string {
	// GovernmentID before ULID so round-scoped queries via
	// begins_with("MSG#{govId}#") are exact.
	if m.GovernmentID != "" {
		return "MSG#" + m.GovernmentID + "#" + m.MessageID
	}
	return ChatMessageKeys.SK(m.MessageID)
}

func (m *ChatMessage) GSIs() map[int]GSIKeyPair { return nil }

func init() {
	RegisterModel(ChatMessageKeys, func() Model { return &ChatMessage{} })
}
