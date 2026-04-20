package models

import (
	"github.com/cannonball10/foundation/schemas/replicant"
)

// VoteKeys provides key construction for the Vote model.
// PK: GAME#{gameId}, SK: VOTE#{governmentId}#{playerId}
// This lets us query all votes for a game, and using begins_with
// "VOTE#{governmentId}#" to fetch just one government's votes.
var VoteKeys = NewKeyBuilder("GAME#", "VOTE#")

// Vote is a single player's ja/nein vote on a proposed government.
// Votes are kept secret from other players until the election resolves.
type Vote struct {
	Timestamps

	GameID       string                  `json:"gameId"`
	GovernmentID string                  `json:"governmentId"`
	PlayerID     string                  `json:"playerId"`
	Choice       replicant.VoteChoice `json:"choice"`
}

// voteSK builds the composite sort key "VOTE#{governmentId}#{playerId}".
func voteSK(governmentID, playerID string) string {
	return VoteKeys.SK(governmentID + "#" + playerID)
}

// NewVote creates a new Vote for the given player on a government.
func NewVote(gameID, governmentID, playerID string, choice replicant.VoteChoice) *Vote {
	return &Vote{
		Timestamps:   NewTimestamps(),
		GameID:       gameID,
		GovernmentID: governmentID,
		PlayerID:     playerID,
		Choice:       choice,
	}
}

func (v *Vote) PK() string { return VoteKeys.PK(v.GameID) }
func (v *Vote) SK() string { return voteSK(v.GovernmentID, v.PlayerID) }

func (v *Vote) GSIs() map[int]GSIKeyPair { return nil }

// IsJa is a convenience for vote counting.
func (v *Vote) IsJa() bool { return v.Choice == replicant.VoteJa }

func init() {
	RegisterModel(VoteKeys, func() Model { return &Vote{} })
}
