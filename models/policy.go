package models

import (
	"fmt"

	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// EnactedPolicyKeys provides key construction for the EnactedPolicy model.
// PK: GAME#{gameId}, SK: POLICY#{sequence}
// Sequence is zero-padded so lexicographic order matches enactment order.
var EnactedPolicyKeys = NewKeyBuilder("GAME#", "POLICY#")

// EnactedPolicy records a single policy that was placed on the board.
// Keeping these as their own records (rather than just a counter on Game)
// gives us an ordered history for replays and end-of-game passports.
type EnactedPolicy struct {
	Timestamps

	GameID       string                  `json:"gameId"`
	Sequence     int                     `json:"sequence"`
	Type         secrethitler.PolicyType `json:"type"`
	GovernmentID string                  `json:"governmentId,omitempty"`
	// TopDeck is true when the policy was force-enacted by the election
	// tracker rather than by a passed government.
	TopDeck bool `json:"topDeck"`
}

// policySK formats the sequence number as a zero-padded SK so that
// lexicographic ordering matches numeric ordering.
func policySK(sequence int) string {
	return EnactedPolicyKeys.SK(fmt.Sprintf("%04d", sequence))
}

// NewEnactedPolicy creates an EnactedPolicy record.
func NewEnactedPolicy(gameID string, sequence int, t secrethitler.PolicyType, governmentID string, topDeck bool) *EnactedPolicy {
	return &EnactedPolicy{
		Timestamps:   NewTimestamps(),
		GameID:       gameID,
		Sequence:     sequence,
		Type:         t,
		GovernmentID: governmentID,
		TopDeck:      topDeck,
	}
}

func (p *EnactedPolicy) PK() string { return EnactedPolicyKeys.PK(p.GameID) }
func (p *EnactedPolicy) SK() string { return policySK(p.Sequence) }

func (p *EnactedPolicy) GSIs() map[int]GSIKeyPair { return nil }

func init() {
	RegisterModel(EnactedPolicyKeys, func() Model { return &EnactedPolicy{} })
}
