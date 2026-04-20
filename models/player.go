package models

import (
	"fmt"

	"github.com/cannonball10/foundation/schemas/secrethitler"
	"github.com/cannonball10/foundation/utils"
)

// PlayerKeys provides key construction for the Player model.
// PK: GAME#{gameId}, SK: PLAYER#{playerId}
// This layout allows fetching all players for a game with a single query.
var PlayerKeys = NewKeyBuilder("GAME#", "PLAYER#")

// PlayerGSI1Keys provides key construction for the Player GSI1
// (list a user's game participations).
// GSI1PK: USER#{userId}, GSI1SK: GAME#{gameId}
var PlayerGSI1Keys = NewKeyBuilder("USER#", "GAME#")

// Player represents a single user's participation in a single game.
// The secret Role and Party must never be exposed to other players
// until the game ends or game-mechanic reveals (investigations,
// Hitler-chancellor win, etc.) occur.
type Player struct {
	Timestamps

	PlayerID    string `json:"playerId"`
	GameID      string `json:"gameId"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`

	// Seat is the 0-indexed position around the table; it defines turn
	// order and is assigned when the game starts.
	Seat int `json:"seat"`

	// Role and Party are assigned at game start. Party tracks what other
	// players would see on an investigation card (Hitler investigates as
	// a fascist but has his own role).
	Role  secrethitler.Role  `json:"role,omitempty"`
	Party secrethitler.Party `json:"party,omitempty"`

	IsHost      bool `json:"isHost"`
	IsAlive     bool `json:"isAlive"`
	IsConnected bool `json:"isConnected"`

	// Country is the world-power delegation this player represents on
	// the Planetary Committee. Assigned at StartGame from a rotating
	// pool keyed on seat so every table has a distinct set. CountryCode
	// is the ISO-3166-1 alpha-2 code used for the passport stamp UI;
	// CountryName is the full display string. Both are empty while the
	// lobby is still accepting joins.
	CountryCode string `json:"countryCode,omitempty"`
	CountryName string `json:"countryName,omitempty"`

	// InvestigatedBySeats tracks which player seats have already used an
	// Investigate Loyalty power on this player, so a given investigator
	// cannot investigate the same target twice in the same game.
	InvestigatedBySeats []int `json:"investigatedBySeats,omitempty"`
}

// NewPlayer creates a new Player. If playerID is empty a ULID is generated.
func NewPlayer(playerID *string, gameID, userID, displayName string, isHost bool) *Player {
	if playerID == nil || *playerID == "" {
		ulid := utils.GenerateULID()
		playerID = &ulid
	}
	return &Player{
		Timestamps:  NewTimestamps(),
		PlayerID:    *playerID,
		GameID:      gameID,
		UserID:      userID,
		DisplayName: displayName,
		IsHost:      isHost,
		IsAlive:     true,
		IsConnected: true,
	}
}

func (p *Player) PK() string { return PlayerKeys.PK(p.GameID) }
func (p *Player) SK() string { return PlayerKeys.SK(p.PlayerID) }

func (p *Player) GSIs() map[int]GSIKeyPair {
	return map[int]GSIKeyPair{
		1: PlayerGSI1Keys.Pair(p.UserID, p.GameID),
	}
}

// AssignRole sets a player's secret role and derives the party.
func (p *Player) AssignRole(role secrethitler.Role) {
	p.Role = role
	p.Party = secrethitler.PartyFor(role)
	p.Touch()
}

// Kill marks the player as executed. The player stays in the game record
// for the game log but cannot vote, nominate, or win.
func (p *Player) Kill() {
	p.IsAlive = false
	p.Touch()
}

// IsRogue returns true if this player was assigned the Hitler role.
func (p *Player) IsRogue() bool {
	return p.Role == secrethitler.RoleRogue
}

// IsSingularity returns true if this player was assigned the
// Singularity role (only dealt when Rules.EnableSingularity is on).
func (p *Player) IsSingularity() bool {
	return p.Role == secrethitler.RoleSingularity
}

// MarkInvestigatedBy records that the given investigator seat has now
// investigated this player.
func (p *Player) MarkInvestigatedBy(investigatorSeat int) {
	for _, s := range p.InvestigatedBySeats {
		if s == investigatorSeat {
			return
		}
	}
	p.InvestigatedBySeats = append(p.InvestigatedBySeats, investigatorSeat)
	p.Touch()
}

// String returns a debug-friendly description of a Player.
func (p *Player) String() string {
	return fmt.Sprintf("Player{%s seat=%d role=%s alive=%t}", p.DisplayName, p.Seat, p.Role, p.IsAlive)
}

func init() {
	RegisterModel(PlayerKeys, func() Model { return &Player{} })
}
