package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// Common engine errors. They're exported so callers (HTTP/WS layer)
// can map to transport-appropriate responses.
var (
	ErrGameNotFound        = errors.New("game not found")
	ErrNotInPhase          = errors.New("action not allowed in current phase")
	ErrNotPresident        = errors.New("actor is not the current president")
	ErrNotChancellor       = errors.New("actor is not the current chancellor")
	ErrNotHost             = errors.New("actor is not the game host")
	ErrIneligibleCandidate = errors.New("candidate is ineligible (term-limited, dead, or self)")
	ErrAlreadyVoted        = errors.New("player already voted")
	ErrPlayerNotFound      = errors.New("player not found")
	ErrInvalidTransition   = errors.New("invalid state transition")
	ErrDeadlineNotReached  = errors.New("phase deadline not reached")
	ErrNotEnoughPlayers    = errors.New("not enough players to start")
	ErrTooManyPlayers      = errors.New("too many players")
)

// loadGame fetches a Game by ID and returns ErrGameNotFound on miss.
func (h *GameHandler) loadGame(ctx context.Context, gameID string) (*models.Game, error) {
	m, err := h.db.Get(ctx, nil, models.GameKeys.Key(gameID))
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrGameNotFound
	}
	return m.(*models.Game), nil
}

// saveGame persists the game.
func (h *GameHandler) saveGame(ctx context.Context, g *models.Game) error {
	g.Touch()
	return h.db.Upsert(ctx, nil, g)
}

// loadPlayers returns all players in the game sorted by seat.
func (h *GameHandler) loadPlayers(ctx context.Context, gameID string) ([]*models.Player, error) {
	sk := "PLAYER#"
	out, err := h.db.Query(ctx, nil, database.QueryInput{
		PartitionKey: models.PlayerKeys.PK(gameID),
		SortKey:      &database.SortKeyCondition{BeginsWith: &sk},
	}, database.QueryOptions{})
	if err != nil {
		return nil, err
	}
	players := make([]*models.Player, 0, len(out.Models))
	for _, m := range out.Models {
		if p, ok := m.(*models.Player); ok {
			players = append(players, p)
		}
	}
	sortPlayersBySeat(players)
	return players, nil
}

// loadGovernment fetches a Government by id within a game.
func (h *GameHandler) loadGovernment(ctx context.Context, gameID, governmentID string) (*models.Government, error) {
	m, err := h.db.Get(ctx, nil, database.Key{
		"PK": models.GovernmentKeys.PK(gameID),
		"SK": models.GovernmentKeys.SK(governmentID),
	})
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrInvalidTransition
	}
	return m.(*models.Government), nil
}

// loadVotesForGovernment returns every vote cast on a given government.
func (h *GameHandler) loadVotesForGovernment(ctx context.Context, gameID, governmentID string) ([]*models.Vote, error) {
	sk := "VOTE#" + governmentID + "#"
	out, err := h.db.Query(ctx, nil, database.QueryInput{
		PartitionKey: models.VoteKeys.PK(gameID),
		SortKey:      &database.SortKeyCondition{BeginsWith: &sk},
	}, database.QueryOptions{})
	if err != nil {
		return nil, err
	}
	votes := make([]*models.Vote, 0, len(out.Models))
	for _, m := range out.Models {
		if v, ok := m.(*models.Vote); ok {
			votes = append(votes, v)
		}
	}
	return votes, nil
}

// setPhase updates the game phase, emits a PhaseChanged event, and sets
// a new deadline. Returns the created GameEvent so callers can chain
// typed payload emissions.
func (h *GameHandler) setPhase(ctx context.Context, g *models.Game, to secrethitler.GamePhase, reason ProgressReason) *models.GameEvent {
	from := g.Phase
	g.Phase = to
	g.PhaseDeadline = h.deadlineFor(to)

	ev := models.NewGameEvent(g.GameID /*type*/, eventTypeForPhase(to) /*actor*/, "")
	deadlineStr := ""
	if g.PhaseDeadline != nil {
		deadlineStr = g.PhaseDeadline.UTC().Format(time.RFC3339)
	}
	payload := PhaseChangedPayload{From: from, To: to, Reason: reason, Deadline: deadlineStr}
	ev.WithData(map[string]any{
		"from":     string(from),
		"to":       string(to),
		"reason":   string(reason),
		"deadline": deadlineStr,
	})
	h.broadcast(ctx, ev, payload)
	return ev
}

// eventTypeForPhase maps a phase to a representative EventType for the
// phase-change announcement.
func eventTypeForPhase(p secrethitler.GamePhase) secrethitler.EventType {
	switch p {
	case secrethitler.PhaseGameOver:
		return secrethitler.EventGameEnded
	default:
		// Generic phase-change events all reuse the election-result
		// EventType which clients treat as "state changed, re-read".
		return secrethitler.EventElectionResult
	}
}

// deadlineFor computes the phase deadline from the configured timeouts.
// Returns nil for phases that do not auto-advance (lobby, game_over).
func (h *GameHandler) deadlineFor(p secrethitler.GamePhase) *time.Time {
	now := h.clock.Now()
	var d time.Duration
	switch p {
	case secrethitler.PhaseNomination:
		d = h.config.NominationTimeout
	case secrethitler.PhaseElection:
		d = h.config.ElectionTimeout
	case secrethitler.PhaseLegislativePresident, secrethitler.PhaseLegislativeChancellor:
		d = h.config.LegislativeTimeout
	case secrethitler.PhaseExecutiveAction:
		d = h.config.ExecutiveTimeout
	case secrethitler.PhaseVetoRequested:
		d = h.config.VetoTimeout
	default:
		return nil
	}
	t := now.Add(d)
	return &t
}

// findPlayerBySeat returns the player seated at the given seat.
func findPlayerBySeat(players []*models.Player, seat int) *models.Player {
	for _, p := range players {
		if p.Seat == seat {
			return p
		}
	}
	return nil
}

// findPlayerByID returns the player matching the given id.
func findPlayerByID(players []*models.Player, playerID string) *models.Player {
	for _, p := range players {
		if p.PlayerID == playerID {
			return p
		}
	}
	return nil
}

// alivePlayers returns only the alive players from the list.
func alivePlayers(players []*models.Player) []*models.Player {
	out := make([]*models.Player, 0, len(players))
	for _, p := range players {
		if p.IsAlive {
			out = append(out, p)
		}
	}
	return out
}

// sortPlayersBySeat sorts players ascending by seat.
func sortPlayersBySeat(players []*models.Player) {
	for i := 1; i < len(players); i++ {
		for j := i; j > 0 && players[j-1].Seat > players[j].Seat; j-- {
			players[j-1], players[j] = players[j], players[j-1]
		}
	}
}

// nextAliveSeat returns the next alive seat after startSeat (wrapping).
func nextAliveSeat(players []*models.Player, startSeat int) int {
	sortPlayersBySeat(players)
	n := len(players)
	for i := 1; i <= n; i++ {
		candidate := players[(indexOfSeat(players, startSeat)+i)%n]
		if candidate.IsAlive {
			return candidate.Seat
		}
	}
	return startSeat
}

// indexOfSeat returns the slice index of the player at seat, or 0 if absent.
func indexOfSeat(players []*models.Player, seat int) int {
	for i, p := range players {
		if p.Seat == seat {
			return i
		}
	}
	return 0
}

// mustPhase returns nil if the game is in the expected phase, else
// ErrNotInPhase wrapped with context.
func mustPhase(g *models.Game, want secrethitler.GamePhase) error {
	if g.Phase != want {
		return fmt.Errorf("%w: want %s, got %s", ErrNotInPhase, want, g.Phase)
	}
	return nil
}
