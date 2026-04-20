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
	g := m.(*models.Game)
	ensureRules(g)
	return g, nil
}

// ensureRules substitutes DefaultRules when the loaded Game has a
// zero or incomplete RulesConfig. Called from EVERY path that hydrates
// a Game from storage (loadGame, lookupByJoinCode, snapshot reads)
// so engine reads of game.Rules.X always see valid values.
//
// Two failure modes this guards against:
//  1. Records that predate the RulesConfig field deserialise with
//     Rules == zero-value (all fields 0). `len(players) >= MaxPlayers`
//     would read as `>= 0` — true for any player count — and every
//     join is rejected.
//  2. Records saved partway through the RulesConfig rollout may have
//     a few fields set but core ones (MaxPlayers, win thresholds) at
//     zero. Per-field defaulting handles that without invalidating a
//     legitimate host-authored override.
func ensureRules(g *models.Game) {
	if g.Rules.IsZero() {
		g.Rules = secrethitler.DefaultRules()
		return
	}
	// Per-field backfill for partial records. Only fill fields that
	// would crash the engine if left at zero; leave fields the host
	// might intentionally set low (e.g. silence chance = 0) alone.
	d := secrethitler.DefaultRules()
	if g.Rules.MinPlayers <= 0 {
		g.Rules.MinPlayers = d.MinPlayers
	}
	if g.Rules.MaxPlayers <= 0 {
		g.Rules.MaxPlayers = d.MaxPlayers
	}
	if g.Rules.HumanProtocolsInDeck <= 0 {
		g.Rules.HumanProtocolsInDeck = d.HumanProtocolsInDeck
	}
	if g.Rules.AIProtocolsInDeck <= 0 {
		g.Rules.AIProtocolsInDeck = d.AIProtocolsInDeck
	}
	if g.Rules.HumanPoliciesToWin <= 0 {
		g.Rules.HumanPoliciesToWin = d.HumanPoliciesToWin
	}
	if g.Rules.AIPoliciesToWin <= 0 {
		g.Rules.AIPoliciesToWin = d.AIPoliciesToWin
	}
	if g.Rules.CodesTransferAt <= 0 {
		g.Rules.CodesTransferAt = d.CodesTransferAt
	}
	if g.Rules.ElectionTrackerLimit <= 0 {
		g.Rules.ElectionTrackerLimit = d.ElectionTrackerLimit
	}
	if g.Rules.VetoUnlockAt <= 0 {
		g.Rules.VetoUnlockAt = d.VetoUnlockAt
	}
	if g.Rules.CablePhaseMode == "" {
		g.Rules.CablePhaseMode = d.CablePhaseMode
	}
	if g.Rules.CablePhaseDurationSec <= 0 {
		g.Rules.CablePhaseDurationSec = d.CablePhaseDurationSec
	}
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

// progressReasonKey is the private context key under which the advance
// path stashes its ProgressReason (timeout / forced / all_voted) so
// downstream setPhase calls can emit it instead of the default
// ReasonAction.
type progressReasonKey struct{}

// ctxWithReason returns ctx carrying the given progression reason.
func ctxWithReason(ctx context.Context, reason ProgressReason) context.Context {
	return context.WithValue(ctx, progressReasonKey{}, reason)
}

// reasonFromCtx returns the stashed progression reason or fallback.
func reasonFromCtx(ctx context.Context, fallback ProgressReason) ProgressReason {
	if v := ctx.Value(progressReasonKey{}); v != nil {
		if r, ok := v.(ProgressReason); ok {
			return r
		}
	}
	return fallback
}

// setPhase updates the game phase, emits a PhaseChanged event, and sets
// a new deadline. Returns the created GameEvent so callers can chain
// typed payload emissions. If the context carries a ProgressReason
// (e.g. the request entered via ForceProgress or TimerExpired) that
// reason takes precedence, so every phase transition caused by a
// forced or timed-out advance stays labelled correctly even when the
// downstream helper (ChancellorEnact, ResolveVeto, etc.) passes
// ReasonAction internally.
func (h *GameHandler) setPhase(ctx context.Context, g *models.Game, to secrethitler.GamePhase, reason ProgressReason) *models.GameEvent {
	from := g.Phase
	g.Phase = to
	g.PhaseDeadline = h.deadlineFor(g, to)

	reason = reasonFromCtx(ctx, reason)

	ev := models.NewGameEvent(g.GameID /*type*/, eventTypeForPhase(to) /*actor*/, "")
	deadlineStr := ""
	if g.PhaseDeadline != nil {
		deadlineStr = g.PhaseDeadline.UTC().Format(time.RFC3339)
	}
	payload := PhaseChangedPayload{
		From:          from,
		To:            to,
		Reason:        reason,
		Deadline:      deadlineStr,
		PresidentSeat: g.PresidentSeat,
	}
	ev.WithData(map[string]any{
		"from":          string(from),
		"to":            string(to),
		"reason":        string(reason),
		"deadline":      deadlineStr,
		"presidentSeat": g.PresidentSeat,
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
// Cable Phase reads its duration from the game's RulesConfig (per-game
// customisable); all other phases pull from the engine-wide Config.
func (h *GameHandler) deadlineFor(g *models.Game, p secrethitler.GamePhase) *time.Time {
	now := h.clock.Now()
	var d time.Duration
	switch p {
	case secrethitler.PhaseNomination:
		d = h.config.NominationTimeout
	case secrethitler.PhaseCablePhase:
		d = time.Duration(g.Rules.CablePhaseDurationSec) * time.Second
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
	if d <= 0 {
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
