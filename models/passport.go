package models

import (
	"time"

	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// PassportKeys provides key construction for the Passport model.
// PK: PASSPORT#{userId}, SK: PASSPORT#{userId}
// A passport is a per-user aggregate: one record per user across all games.
var PassportKeys = NewKeyBuilder("PASSPORT#", "PASSPORT#")

// PassportEntryKeys provides key construction for per-game entries on a
// user's passport.
// PK: PASSPORT#{userId}, SK: ENTRY#{gameId}
// Entries are the chronological list of games a user has played.
var PassportEntryKeys = NewKeyBuilder("PASSPORT#", "ENTRY#")

// Passport is the aggregate cross-game stat sheet for a single user.
// Counters are updated when a game ends; the per-game ledger is stored
// as separate PassportEntry records under the same PK.
type Passport struct {
	Timestamps

	UserID      string `json:"userId"`
	GamesPlayed int    `json:"gamesPlayed"`
	GamesWon    int    `json:"gamesWon"`

	// Wins broken down by the role the user played during that game.
	WinsAsLiberal int `json:"winsAsLiberal"`
	WinsAsFascist int `json:"winsAsFascist"`
	WinsAsHitler  int `json:"winsAsHitler"`

	// Role-appearance counts (how often the user was dealt each role).
	TimesLiberal int `json:"timesLiberal"`
	TimesFascist int `json:"timesFascist"`
	TimesHitler  int `json:"timesHitler"`

	// Additional notable stats.
	TimesExecuted          int        `json:"timesExecuted"`
	TimesElectedPresident  int        `json:"timesElectedPresident"`
	TimesElectedChancellor int        `json:"timesElectedChancellor"`
	LastPlayedAt           *time.Time `json:"lastPlayedAt,omitempty"`
}

// NewPassport creates an empty Passport for a user.
func NewPassport(userID string) *Passport {
	return &Passport{
		Timestamps: NewTimestamps(),
		UserID:     userID,
	}
}

func (p *Passport) PK() string { return PassportKeys.PK(p.UserID) }
func (p *Passport) SK() string { return PassportKeys.SK(p.UserID) }

func (p *Passport) GSIs() map[int]GSIKeyPair { return nil }

// RecordGame increments the passport for a completed game's outcome.
// role is the role the user played; won indicates whether their party
// won. The caller is responsible for also writing a PassportEntry.
func (p *Passport) RecordGame(role secrethitler.Role, won bool, endedAt time.Time) {
	p.GamesPlayed++
	switch role {
	case secrethitler.RoleLiberal:
		p.TimesLiberal++
		if won {
			p.WinsAsLiberal++
		}
	case secrethitler.RoleFascist:
		p.TimesFascist++
		if won {
			p.WinsAsFascist++
		}
	case secrethitler.RoleHitler:
		p.TimesHitler++
		if won {
			p.WinsAsHitler++
		}
	}
	if won {
		p.GamesWon++
	}
	p.LastPlayedAt = &endedAt
	p.Touch()
}

// PassportEntry is the record of a single game on a user's passport.
// It is written once a game ends so the player can review their history.
type PassportEntry struct {
	Timestamps

	UserID       string                    `json:"userId"`
	GameID       string                    `json:"gameId"`
	PlayerID     string                    `json:"playerId"`
	Role         secrethitler.Role         `json:"role"`
	Party        secrethitler.Party        `json:"party"`
	Won          bool                      `json:"won"`
	WinCondition secrethitler.WinCondition `json:"winCondition,omitempty"`
	Seat         int                       `json:"seat"`
	PlayerCount  int                       `json:"playerCount"`
	StartedAt    time.Time                 `json:"startedAt"`
	EndedAt      time.Time                 `json:"endedAt"`
}

// NewPassportEntry creates a PassportEntry for a completed game.
func NewPassportEntry(userID, gameID, playerID string, role secrethitler.Role, party secrethitler.Party, won bool, winCondition secrethitler.WinCondition, seat, playerCount int, startedAt, endedAt time.Time) *PassportEntry {
	return &PassportEntry{
		Timestamps:   NewTimestamps(),
		UserID:       userID,
		GameID:       gameID,
		PlayerID:     playerID,
		Role:         role,
		Party:        party,
		Won:          won,
		WinCondition: winCondition,
		Seat:         seat,
		PlayerCount:  playerCount,
		StartedAt:    startedAt,
		EndedAt:      endedAt,
	}
}

func (e *PassportEntry) PK() string { return PassportEntryKeys.PK(e.UserID) }
func (e *PassportEntry) SK() string { return PassportEntryKeys.SK(e.GameID) }

func (e *PassportEntry) GSIs() map[int]GSIKeyPair { return nil }

func init() {
	RegisterModel(PassportKeys, func() Model { return &Passport{} })
	RegisterModel(PassportEntryKeys, func() Model { return &PassportEntry{} })
}
