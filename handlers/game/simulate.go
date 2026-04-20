package game

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/replicant"
)

// simCableBodies is the bot cable corpus. Deliberately mixed in tone
// so the LLM ranker has both "innocuous" and "suspicious" entries to
// score against. Not meant to be clever — just enough traffic to keep
// the Cable Phase visually alive in autonomous demos.
var simCableBodies = []string{
	"I move we adjourn for a brief recess.",
	"The prior tribunal's findings still give me pause.",
	"vote nein on this government. too risky.",
	"Anyone else uncomfortable with Delegate 03's pattern here.",
	"No objection from this delegation.",
	"we need to flip jordan before the third enactment",
	"Can we get a clarification on the last executive order.",
	"The Committee's silence is itself a signal.",
	"Delegate 05 is behaving exactly like the last replicant.",
	"Fine. Pass it. We'll see what the board does.",
}

// SimulationConfig tunes a bot-driven playthrough of an existing lobby.
// All zero values map to sensible defaults so callers can pass an empty
// struct for a reasonable demo.
type SimulationConfig struct {
	// Players is the total seat count, including the host. Clamped to
	// [replicant.MinPlayers, replicant.MaxPlayers]. Default 7.
	Players int
	// StepDelay paces engine actions so a human viewer can follow along.
	// Default 1500ms.
	StepDelay time.Duration
	// StartDelay waits after the lobby exists before bots begin joining,
	// giving the host's SSE stream time to attach so early envelopes
	// aren't missed. Default 2s.
	StartDelay time.Duration
	// MaxSteps caps the main action loop; the simulator exits with a
	// warn log if a game doesn't finish in this many actions. Default 500.
	MaxSteps int
	// Seed deterministically drives bot decisions when non-zero. Default
	// is time-based so successive demos don't play out identically.
	Seed uint64
	// HumanSeats reserves seats for real players to occupy via the join
	// flow. Bots fill only (Players - HumanSeats) seats, the simulator
	// does NOT call StartGame (the host presses Start from the board
	// once humans are in), and during play the simulator never acts on
	// behalf of non-bot players. Default 0 = fully autonomous demo.
	HumanSeats int
}

// SimulateGame attaches bot players to an existing lobby and plays it
// to completion. Intended for dev demos: drop it behind a feature flag
// so the HTTP CreateGame handler can spin one up per new board.
//
// Behaviour depends on cfg.HumanSeats:
//
//   - 0: fully autonomous. Bots fill every seat (including the host's
//     if they're seated), the simulator calls StartGame itself, and
//     plays every turn on every player's behalf. Classic demo mode.
//   - >0: mixed mode. Bots fill (Players-HumanSeats) seats, leaving
//     room for real players to join via the normal join flow. The
//     simulator does not call StartGame — the host presses Start from
//     the board once the room is full. During play, turns owned by
//     humans are skipped (the simulator re-polls each StepDelay and
//     only acts when the actor is a bot).
//
// Errors are logged and the goroutine returns; a failing simulation
// must never panic the server.
func (h *GameHandler) SimulateGame(ctx context.Context, gameID string, cfg SimulationConfig) {
	cfg = applySimDefaults(cfg)
	log := slog.With("sim", shortID(gameID))

	g, err := h.loadGame(ctx, gameID)
	if err != nil {
		log.Error("loadGame failed", "err", err)
		return
	}
	if g.Status != replicant.GameStatusLobby {
		log.Warn("game not in lobby; skipping simulation", "status", g.Status)
		return
	}

	// Simulated demos turn on the features that require a narrator or
	// larger lobbies: Cable Phase every round so the LLM rank/leak path
	// is exercised, and Singularity seating for 6+ player games so the
	// kingmaker win condition is reachable. The flags are no-ops for
	// games created outside SimulateGame — production CreateGame stamps
	// DefaultRules and never calls us.
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	if cfg.Players >= 6 {
		g.Rules.EnableSingularity = true
	}
	if err := h.saveGame(ctx, g); err != nil {
		log.Error("failed to apply simulation rule overrides", "err", err)
		return
	}
	log.Info("sim rules applied",
		"cable", g.Rules.CablePhaseMode,
		"singularity", g.Rules.EnableSingularity,
		"duration_sec", g.Rules.CablePhaseDurationSec)

	// Let the host's SSE subscription attach so early join envelopes
	// land on the wire rather than in /dev/null.
	if !sleep(ctx, cfg.StartDelay) {
		return
	}

	rng := NewSeededRNG(cfg.Seed)

	// Top up seats to (Players - HumanSeats). HumanSeats=0 means fill
	// every chair with a bot and auto-start, the classic demo flow.
	botTarget := cfg.Players - cfg.HumanSeats
	if botTarget < 0 {
		botTarget = 0
	}
	existing, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		log.Error("loadPlayers failed", "err", err)
		return
	}
	for i := len(existing); i < botTarget; i++ {
		uid := botUserID(gameID, i)
		name := fmt.Sprintf("Bot %d", i)
		if _, _, err := h.JoinGame(ctx, g.JoinCode, uid, name); err != nil {
			log.Error("JoinGame failed", "i", i, "err", err)
			return
		}
		// Cascade joins quickly rather than at full StepDelay.
		if !sleep(ctx, cfg.StepDelay/3) {
			return
		}
	}

	if cfg.HumanSeats > 0 {
		// Humans expected: don't call StartGame ourselves. The host
		// clicks Start from the board once real players have joined.
		// Poll the lobby until status changes so we can kick off the
		// action loop as soon as the game begins.
		log.Info("waiting for host to start", "bots", botTarget, "humans_expected", cfg.HumanSeats)
		for {
			if !sleep(ctx, 2*time.Second) {
				return
			}
			gme, err := h.loadGame(ctx, gameID)
			if err != nil {
				log.Error("loadGame (waiting for start) failed", "err", err)
				return
			}
			if gme.Status != replicant.GameStatusLobby {
				break
			}
		}
	} else {
		if _, err := h.StartGame(ctx, gameID, g.HostUserID); err != nil {
			log.Error("StartGame failed", "err", err)
			return
		}
	}

	// Per-step trace so we can diagnose hangs in the wild. When a game
	// gets stuck, the log tells us exactly which phase the sim is
	// wedged on and who it's waiting for.
	var lastPhase replicant.GamePhase
	for step := 0; step < cfg.MaxSteps; step++ {
		if !sleep(ctx, cfg.StepDelay) {
			return
		}
		game, err := h.loadGame(ctx, gameID)
		if err != nil {
			log.Error("loadGame failed mid-game", "step", step, "err", err)
			return
		}
		if game.Status == replicant.GameStatusCompleted {
			log.Info("simulation complete",
				"winner", game.Winner, "condition", game.WinCondition,
				"human", game.HumanPoliciesEnacted,
				"ai", game.AIPoliciesEnacted)
			return
		}
		if game.Phase != lastPhase {
			log.Info("sim phase", "step", step, "phase", game.Phase, "round", game.Round,
				"pres_seat", game.PresidentSeat)
			lastPhase = game.Phase
		}
		players, err := h.loadPlayers(ctx, gameID)
		if err != nil {
			log.Error("loadPlayers failed", "err", err)
			return
		}
		if err := h.stepSim(ctx, game, players, rng); err != nil {
			log.Error("step failed", "phase", game.Phase, "step", step, "err", err)
			return
		}
	}
	log.Warn("hit MaxSteps without completing", "max", cfg.MaxSteps)
}

// stepSim advances the game by one engine call chosen for the current
// phase. Choices are random among legal options — no strategy, because
// the goal is to exercise the UI, not win.
//
// Turns owned by human players (userID without the "sim-" prefix) are
// left alone; the outer loop sleeps and re-checks, effectively polling
// until the human acts via the regular HTTP endpoints.
func (h *GameHandler) stepSim(ctx context.Context, game *models.Game, players []*models.Player, rng *SeededRNG) error {
	switch game.Phase {
	case replicant.PhaseNomination:
		president := findPlayerBySeat(players, game.PresidentSeat)
		if president == nil {
			return fmt.Errorf("no president at seat %d", game.PresidentSeat)
		}
		if !isBotUser(president.UserID) {
			return nil // human's nomination — wait
		}
		chancellor := pickSimChancellor(game, players, president, rng)
		if chancellor == nil {
			return fmt.Errorf("no eligible chancellor")
		}
		_, err := h.NominateChancellor(ctx, game.GameID, president.PlayerID, chancellor.PlayerID)
		return err

	case replicant.PhaseCablePhase:
		// Each living bot submits one short canned cable per phase so
		// the LLM ranker has something to score (and the host can see
		// the leak UI exercised). Humans in the room still compose
		// via the mobile app. Once every bot has dropped its cable,
		// force-progress to the election vote without waiting for
		// the full cable timer — keeps sim tempo brisk.
		for _, p := range alivePlayers(players) {
			if !isBotUser(p.UserID) {
				continue
			}
			body := simCableBodies[rng.IntN(len(simCableBodies))]
			if err := h.SendChat(ctx, game.GameID, p.PlayerID, ChannelCable, body); err != nil {
				// Log but keep going — one bot's failed submission
				// shouldn't stall the phase.
				slog.Warn("sim cable submission failed", "err", err, "player", p.PlayerID)
			}
		}
		return h.ForceProgress(ctx, game.GameID, game.HostUserID)

	case replicant.PhaseElection:
		// Cast a ja/nein vote for every bot that hasn't voted yet.
		// Humans vote themselves through POST /player/vote; we never
		// cast on their behalf. The engine resolves the election inline
		// the moment the final alive player votes, so we re-check phase
		// after every cast to bail out on the vote that tips it.
		existingVotes, err := h.loadVotesForGovernment(ctx, game.GameID, game.CurrentGovernmentID)
		if err != nil {
			return err
		}
		voted := make(map[string]bool, len(existingVotes))
		for _, v := range existingVotes {
			voted[v.PlayerID] = true
		}
		for _, p := range alivePlayers(players) {
			if !isBotUser(p.UserID) || voted[p.PlayerID] {
				continue
			}
			choice := replicant.VoteJa
			if rng.IntN(5) == 0 { // ~20% nein, enough to see the tracker advance occasionally
				choice = replicant.VoteNein
			}
			if err := h.CastVote(ctx, game.GameID, p.PlayerID, choice); err != nil {
				return err
			}
			g2, err := h.loadGame(ctx, game.GameID)
			if err != nil {
				return err
			}
			if g2.Phase != replicant.PhaseElection {
				break
			}
		}
		return nil

	case replicant.PhaseLegislativePresident:
		president := findPlayerBySeat(players, game.PresidentSeat)
		if president == nil {
			return fmt.Errorf("no president at seat %d", game.PresidentSeat)
		}
		if !isBotUser(president.UserID) {
			return nil
		}
		gov, err := h.loadGovernment(ctx, game.GameID, game.CurrentGovernmentID)
		if err != nil {
			return err
		}
		if len(gov.DrawnPolicies) == 0 {
			return fmt.Errorf("legislative_president with empty DrawnPolicies")
		}
		idx := rng.IntN(len(gov.DrawnPolicies))
		return h.PresidentDiscard(ctx, game.GameID, president.PlayerID, idx)

	case replicant.PhaseLegislativeChancellor:
		if game.ChancellorSeat == nil {
			return fmt.Errorf("legislative_chancellor with no chancellor seat")
		}
		chancellor := findPlayerBySeat(players, *game.ChancellorSeat)
		if chancellor == nil {
			return fmt.Errorf("no chancellor at seat %d", *game.ChancellorSeat)
		}
		if !isBotUser(chancellor.UserID) {
			return nil
		}
		gov, err := h.loadGovernment(ctx, game.GameID, game.CurrentGovernmentID)
		if err != nil {
			return err
		}
		if len(gov.ChancellorOptions) == 0 {
			return fmt.Errorf("legislative_chancellor with empty ChancellorOptions")
		}
		idx := rng.IntN(len(gov.ChancellorOptions))
		return h.ChancellorEnact(ctx, game.GameID, chancellor.PlayerID, idx)

	case replicant.PhaseVetoRequested:
		// Reject by default so the chancellor is forced to enact and the
		// game keeps progressing. Always-accepting would just burn the
		// tracker; demos are more interesting with policies hitting the board.
		president := findPlayerBySeat(players, game.PresidentSeat)
		if president == nil {
			return fmt.Errorf("no president at seat %d", game.PresidentSeat)
		}
		if !isBotUser(president.UserID) {
			return nil
		}
		return h.ResolveVeto(ctx, game.GameID, president.PlayerID, false)

	case replicant.PhaseExecutiveAction:
		president := findPlayerBySeat(players, game.PresidentSeat)
		if president == nil {
			return fmt.Errorf("no president at seat %d", game.PresidentSeat)
		}
		if !isBotUser(president.UserID) {
			return nil
		}
		target := pickSimExecutiveTarget(game, players, president)
		if target == nil {
			return fmt.Errorf("no target for %s", game.PendingActionType)
		}
		return h.ExecuteAction(ctx, game.GameID, president.PlayerID, target.PlayerID)
	}
	return fmt.Errorf("unhandled phase %q", game.Phase)
}

// isBotUser reports whether a userID belongs to a simulated player. All
// bots the simulator spawns have userIDs of the form "sim-<shortID>-<n>".
// Treating the prefix as the signal keeps the engine package clean: the
// GameHandler has no notion of "bot"; only the simulator does.
func isBotUser(userID string) bool {
	return strings.HasPrefix(userID, "sim-")
}

func applySimDefaults(cfg SimulationConfig) SimulationConfig {
	if cfg.Players <= 0 {
		cfg.Players = 7
	}
	if cfg.Players < replicant.MinPlayers {
		cfg.Players = replicant.MinPlayers
	}
	if cfg.Players > replicant.MaxPlayers {
		cfg.Players = replicant.MaxPlayers
	}
	if cfg.StepDelay <= 0 {
		cfg.StepDelay = 1500 * time.Millisecond
	}
	if cfg.StartDelay <= 0 {
		cfg.StartDelay = 2 * time.Second
	}
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 500
	}
	if cfg.Seed == 0 {
		cfg.Seed = uint64(time.Now().UnixNano())
	}
	if cfg.HumanSeats < 0 {
		cfg.HumanSeats = 0
	}
	if cfg.HumanSeats > cfg.Players-replicant.MinPlayers+1 {
		// Leave at least enough bots for a legal game. Clamp rather
		// than error — dev-only switch, easier to reason about.
		cfg.HumanSeats = cfg.Players - replicant.MinPlayers + 1
		if cfg.HumanSeats < 0 {
			cfg.HumanSeats = 0
		}
	}
	return cfg
}

// sleep returns false if ctx is cancelled during the wait.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func botUserID(gameID string, i int) string {
	return fmt.Sprintf("sim-%s-%d", shortID(gameID), i)
}

func shortID(id string) string {
	return id[:min(8, len(id))]
}

// pickSimChancellor picks a uniformly random eligible chancellor. Uses
// the same rule set as the engine so we never invite ErrIneligibleCandidate.
// Logs a summary of the candidate pool + selection so the post-mortem
// on a suspicious nomination (e.g. "Bot 4 picked Bot 0 but Bot 0 is dead")
// can trace whether the simulator or the engine is responsible.
func pickSimChancellor(game *models.Game, players []*models.Player, president *models.Player, rng *SeededRNG) *models.Player {
	aliveCount := len(alivePlayers(players))
	cand := make([]*models.Player, 0, len(players))
	dead := 0
	for _, p := range players {
		if !p.IsAlive {
			dead++
			continue
		}
		if p.PlayerID == president.PlayerID {
			continue
		}
		if game.PreviousChancellorSeat != nil && *game.PreviousChancellorSeat == p.Seat {
			continue
		}
		if aliveCount > 5 && game.PreviousPresidentSeat != nil && *game.PreviousPresidentSeat == p.Seat {
			continue
		}
		cand = append(cand, p)
	}
	if len(cand) == 0 {
		slog.Warn("pickSimChancellor: no candidates",
			"gameId", shortID(game.GameID), "presSeat", president.Seat,
			"total", len(players), "dead", dead, "aliveCount", aliveCount)
		return nil
	}
	pick := cand[rng.IntN(len(cand))]
	slog.Debug("pickSimChancellor",
		"gameId", shortID(game.GameID), "presSeat", president.Seat,
		"pickSeat", pick.Seat, "pickAlive", pick.IsAlive,
		"candidates", len(cand), "dead", dead)
	return pick
}

// pickSimExecutiveTarget picks the first legal target, skipping
// already-investigated seats for Investigate. First-legal (not random)
// is fine here — the power effect is identical regardless of which
// target is chosen.
func pickSimExecutiveTarget(game *models.Game, players []*models.Player, president *models.Player) *models.Player {
	for _, p := range alivePlayers(players) {
		if p.PlayerID == president.PlayerID {
			continue
		}
		if game.PendingActionType == replicant.ActionInvestigateLoyalty && seatAlreadyInvestigated(p, president.Seat) {
			continue
		}
		return p
	}
	return nil
}
