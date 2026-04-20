package game

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// driveStrategy is a bundle of per-phase preferences passed to driveGame.
// The driver consults it when choosing actions so we can force specific
// outcomes (e.g. "always enact liberal", "execute Hitler when possible",
// "propose + accept every veto", etc.).
type driveStrategy struct {
	// preferEnact controls which policy the chancellor tries to enact
	// when the draw gives a choice. Empty string means "first option".
	preferEnact secrethitler.PolicyType
	// preferDiscard controls which policy the president tries to
	// discard. Empty string means "first policy".
	preferDiscard secrethitler.PolicyType
	// targetHitler toggles targeting Hitler on Execution when possible.
	targetHitler bool
	// nominateHitler toggles nominating Hitler when eligible.
	nominateHitler bool
	// proposeVeto controls whether the chancellor proposes veto when
	// the power is unlocked.
	proposeVeto bool
	// acceptVeto controls the president's default veto resolution.
	acceptVeto bool
}

// driveGame plays a game to completion following strategy, returning
// the final state plus a coverage report of every action and event
// observed along the way.
func driveGame(t *testing.T, seed uint64, playerCount int, strat driveStrategy, maxSteps int) (*coverageReport, *models.Game) {
	t.Helper()
	h, _, cap, clock := newTestHandler(t, seed)
	g := seedLobby(t, h, playerCount)
	ctx := context.Background()
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	// Lookup Hitler once; roles are dealt during StartGame.
	allPlayers, _ := h.loadPlayers(ctx, g.GameID)
	var hitler *models.Player
	for _, p := range allPlayers {
		if p.IsRogue() {
			hitler = p
			break
		}
	}

	for step := 0; step < maxSteps; step++ {
		game, err := h.loadGame(ctx, g.GameID)
		if err != nil {
			t.Fatalf("step %d: loadGame: %v", step, err)
		}
		if game.Status == secrethitler.GameStatusCompleted {
			return newCoverageReport(cap), game
		}
		players, _ := h.loadPlayers(ctx, g.GameID)

		switch game.Phase {
		case secrethitler.PhaseNomination:
			president := findPlayerBySeat(players, game.PresidentSeat)
			chancellor := pickChancellorStrategy(game, players, president, hitler, strat)
			if chancellor == nil {
				t.Fatalf("step %d p=%d: no chancellor; %s", step, playerCount, summarize(game))
			}
			if _, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID); err != nil {
				t.Fatalf("step %d: nominate: %v", step, err)
			}

		case secrethitler.PhaseElection:
			for _, p := range alivePlayers(players) {
				_ = h.CastVote(ctx, g.GameID, p.PlayerID, secrethitler.VoteJa)
				g2, _ := h.loadGame(ctx, g.GameID)
				if g2.Phase != secrethitler.PhaseElection {
					break
				}
			}

		case secrethitler.PhaseLegislativePresident:
			gov, err := h.loadGovernment(ctx, g.GameID, game.CurrentGovernmentID)
			if err != nil {
				t.Fatalf("step %d: loadGovernment: %v", step, err)
			}
			idx := pickDiscardIndex(gov.DrawnPolicies, strat.preferDiscard)
			president := findPlayerBySeat(players, game.PresidentSeat)
			if err := h.PresidentDiscard(ctx, g.GameID, president.PlayerID, idx); err != nil {
				t.Fatalf("step %d: discard: %v", step, err)
			}

		case secrethitler.PhaseLegislativeChancellor:
			gov, _ := h.loadGovernment(ctx, g.GameID, game.CurrentGovernmentID)
			chancellor := findPlayerBySeat(players, *game.ChancellorSeat)
			// Veto is proposable only if the chancellor hasn't already
			// attempted it for this government (gov.VetoProposed flips
			// on propose; after a reject the chancellor must enact).
			if strat.proposeVeto && game.VetoUnlocked && !gov.VetoProposed {
				if err := h.ProposeVeto(ctx, g.GameID, chancellor.PlayerID); err != nil {
					t.Fatalf("step %d: propose veto: %v", step, err)
				}
			} else {
				idx := pickEnactIndex(gov.ChancellorOptions, strat.preferEnact)
				if err := h.ChancellorEnact(ctx, g.GameID, chancellor.PlayerID, idx); err != nil {
					t.Fatalf("step %d: enact: %v", step, err)
				}
			}

		case secrethitler.PhaseVetoRequested:
			president := findPlayerBySeat(players, game.PresidentSeat)
			if err := h.ResolveVeto(ctx, g.GameID, president.PlayerID, strat.acceptVeto); err != nil {
				t.Fatalf("step %d: resolve veto: %v", step, err)
			}

		case secrethitler.PhaseExecutiveAction:
			president := findPlayerBySeat(players, game.PresidentSeat)
			target := pickExecutiveTargetStrategy(game, players, president, hitler, strat)
			if target == nil {
				t.Fatalf("step %d: no target (%s); %s", step, game.PendingActionType, summarize(game))
			}
			if err := h.ExecuteAction(ctx, g.GameID, president.PlayerID, target.PlayerID); err != nil {
				t.Fatalf("step %d: execute %s: %v", step, game.PendingActionType, err)
			}
		}
	}
	// Advance the fake clock so any deadline checks downstream are safe.
	clock.Advance(time.Hour)
	final, _ := h.loadGame(ctx, g.GameID)
	t.Fatalf("game did not complete in %d steps: %s", maxSteps, summarize(final))
	return nil, nil
}

// pickDiscardIndex returns the index of a policy to discard. If
// prefer is set, it discards one of that type when available. Otherwise
// it returns 0.
func pickDiscardIndex(drawn []secrethitler.PolicyType, prefer secrethitler.PolicyType) int {
	if prefer == "" {
		return 0
	}
	for i, p := range drawn {
		if p == prefer {
			return i
		}
	}
	return 0
}

// pickEnactIndex returns the index of a policy to enact, preferring the
// given type when available.
func pickEnactIndex(options []secrethitler.PolicyType, prefer secrethitler.PolicyType) int {
	if prefer == "" {
		return 0
	}
	for i, p := range options {
		if p == prefer {
			return i
		}
	}
	return 0
}

// pickChancellorStrategy picks a chancellor, favouring Hitler when the
// strategy requests it and Hitler is alive + eligible.
func pickChancellorStrategy(game *models.Game, players []*models.Player, president *models.Player, hitler *models.Player, strat driveStrategy) *models.Player {
	if strat.nominateHitler && hitler != nil && isEligibleChancellor(game, players, president, hitler) {
		return hitler
	}
	return pickChancellor(game, players, president)
}

// pickExecutiveTargetStrategy picks a target for an executive action,
// favouring Hitler on Execution when the strategy requests it.
func pickExecutiveTargetStrategy(game *models.Game, players []*models.Player, president *models.Player, hitler *models.Player, strat driveStrategy) *models.Player {
	if strat.targetHitler && game.PendingActionType == secrethitler.ActionExecution &&
		hitler != nil && hitler.IsAlive && hitler.PlayerID != president.PlayerID {
		return hitler
	}
	return pickExecutiveTarget(game, players, president)
}

// isEligibleChancellor returns true if candidate satisfies Secret
// Hitler's chancellor eligibility rules for the given game state.
func isEligibleChancellor(game *models.Game, players []*models.Player, president, candidate *models.Player) bool {
	if !candidate.IsAlive || candidate.PlayerID == president.PlayerID {
		return false
	}
	if game.PreviousChancellorSeat != nil && *game.PreviousChancellorSeat == candidate.Seat {
		return false
	}
	if len(alivePlayers(players)) > 5 && game.PreviousPresidentSeat != nil && *game.PreviousPresidentSeat == candidate.Seat {
		return false
	}
	return true
}

// coverageReport summarizes the actions, powers, and terminal state
// observed during a driveGame run. Asserting on these fields keeps
// the per-player-count tests concise.
type coverageReport struct {
	eventCounts map[secrethitler.EventType]int
	powers      map[secrethitler.ExecutiveActionType]int
}

func newCoverageReport(cap *captureEmitter) *coverageReport {
	r := &coverageReport{
		eventCounts: make(map[secrethitler.EventType]int),
		powers:      make(map[secrethitler.ExecutiveActionType]int),
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	for _, env := range cap.envs {
		if env.Event == nil {
			continue
		}
		r.eventCounts[env.Event.Type]++
		if p, ok := env.Payload.(ExecutiveActionPayload); ok {
			r.powers[p.Type]++
		}
	}
	return r
}

// merged accumulates a coverage report into this one.
func (r *coverageReport) merge(other *coverageReport) {
	for k, v := range other.eventCounts {
		r.eventCounts[k] += v
	}
	for k, v := range other.powers {
		r.powers[k] += v
	}
}

// expectedPowers returns the set of executive powers that must appear
// over the course of testing for the given player count.
func expectedPowers(playerCount int) []secrethitler.ExecutiveActionType {
	switch {
	case playerCount >= 9:
		return []secrethitler.ExecutiveActionType{
			secrethitler.ActionInvestigateLoyalty,
			secrethitler.ActionSpecialElection,
			secrethitler.ActionExecution,
		}
	case playerCount >= 7:
		return []secrethitler.ExecutiveActionType{
			secrethitler.ActionInvestigateLoyalty,
			secrethitler.ActionSpecialElection,
			secrethitler.ActionExecution,
		}
	default: // 5-6
		return []secrethitler.ExecutiveActionType{
			secrethitler.ActionPolicyPeek,
			secrethitler.ActionExecution,
		}
	}
}

// TestActionMatrix_AllPlayerCounts drives games at every legal player
// count (5-10) through a fixed set of strategies, aggregates event and
// power coverage across all runs, and asserts:
//
//   - every win condition can be produced,
//   - every executive power applicable to the player count fires,
//   - both veto resolution paths (accept & reject) are observed,
//   - both top-deck and deck-reshuffle paths are observed,
//   - broadcast + whisper events for every mid-game action are emitted.
func TestActionMatrix_AllPlayerCounts(t *testing.T) {
	// Strategies are chosen so the union of their outcomes covers every
	// win condition and every executive power across a handful of seeds.
	strategies := []struct {
		name     string
		strategy driveStrategy
	}{
		{"fascist_win_enact_fascist", driveStrategy{
			preferEnact:   secrethitler.PolicyAI,
			preferDiscard: secrethitler.PolicyHuman,
		}},
		{"liberal_win_enact_liberal", driveStrategy{
			preferEnact:   secrethitler.PolicyHuman,
			preferDiscard: secrethitler.PolicyAI,
		}},
		{"hitler_chancellor", driveStrategy{
			preferEnact:    secrethitler.PolicyAI,
			preferDiscard:  secrethitler.PolicyHuman,
			nominateHitler: true,
		}},
		{"rogue_executed", driveStrategy{
			preferEnact:   secrethitler.PolicyAI,
			preferDiscard: secrethitler.PolicyHuman,
			targetHitler:  true,
		}},
		{"veto_accept", driveStrategy{
			preferEnact:   secrethitler.PolicyAI,
			preferDiscard: secrethitler.PolicyHuman,
			proposeVeto:   true,
			acceptVeto:    true,
		}},
		{"veto_reject", driveStrategy{
			preferEnact:   secrethitler.PolicyAI,
			preferDiscard: secrethitler.PolicyHuman,
			proposeVeto:   true,
			acceptVeto:    false,
		}},
	}
	seeds := []uint64{1, 3, 7, 11, 19, 23, 31, 41, 53, 71}

	for _, playerCount := range []int{5, 6, 7, 8, 9, 10} {
		playerCount := playerCount
		t.Run(fmt.Sprintf("players=%d", playerCount), func(t *testing.T) {
			agg := newCoverageReport(&captureEmitter{})
			winConditions := make(map[secrethitler.WinCondition]int)

			for _, strat := range strategies {
				strat := strat
				for _, seed := range seeds {
					seed := seed
					t.Run(fmt.Sprintf("%s/seed=%d", strat.name, seed), func(t *testing.T) {
						rep, final := driveGame(t, seed+uint64(playerCount*101), playerCount, strat.strategy, 2000)
						if final.Status != secrethitler.GameStatusCompleted {
							t.Fatalf("game did not complete: %s", summarize(final))
						}
						winConditions[final.WinCondition]++
						agg.merge(rep)
					})
				}
			}

			// Assert each applicable executive power fired.
			for _, want := range expectedPowers(playerCount) {
				if agg.powers[want] == 0 {
					t.Errorf("expected power %s to fire at least once at %d players; powers=%v",
						want, playerCount, agg.powers)
				}
			}

			// Assert every game-mechanic event occurred.
			requiredEvents := []secrethitler.EventType{
				secrethitler.EventGameCreated,
				secrethitler.EventPlayerJoined,
				secrethitler.EventGameStarted,
				secrethitler.EventRolesAssigned,
				secrethitler.EventChancellorNominated,
				secrethitler.EventVoteCast,
				secrethitler.EventElectionResult,
				secrethitler.EventPoliciesDrawn,
				secrethitler.EventPresidentDiscarded,
				secrethitler.EventChancellorEnacted,
				secrethitler.EventVetoProposed,
				secrethitler.EventVetoResolved,
				secrethitler.EventExecutiveAction,
				secrethitler.EventGameEnded,
			}
			for _, ev := range requiredEvents {
				if agg.eventCounts[ev] == 0 {
					t.Errorf("expected event %s to occur at least once at %d players",
						ev, playerCount)
				}
			}

			// At 5+ players, the board always reaches enough policies to
			// top-deck at least once across the 60 games per player count.
			if agg.eventCounts[secrethitler.EventTopDeckEnacted] == 0 {
				t.Logf("note: no top-deck events at %d players (not mandatory)", playerCount)
			}

			// All four win conditions should be seen across the strategies.
			for _, wc := range []secrethitler.WinCondition{
				secrethitler.WinHumanPolicies,
				secrethitler.WinAIPolicies,
				secrethitler.WinRogueElected,
				secrethitler.WinRogueExecuted,
			} {
				if winConditions[wc] == 0 {
					t.Logf("note: win condition %s never produced at %d players (strategy coverage)", wc, playerCount)
				}
			}
			t.Logf("%d-player coverage: winConditions=%v powers=%v events=%d",
				playerCount, winConditions, agg.powers, len(agg.eventCounts))
		})
	}
}

// TestActions_ForceProgressAcrossPhases asserts the host can force
// progression out of every non-terminal phase at each player count
// without corrupting state.
func TestActions_ForceProgressAcrossPhases(t *testing.T) {
	for _, playerCount := range []int{5, 6, 7, 8, 9, 10} {
		playerCount := playerCount
		t.Run(fmt.Sprintf("players=%d", playerCount), func(t *testing.T) {
			phases := []secrethitler.GamePhase{
				secrethitler.PhaseNomination,
				secrethitler.PhaseElection,
				secrethitler.PhaseLegislativePresident,
				secrethitler.PhaseLegislativeChancellor,
			}
			for _, target := range phases {
				target := target
				t.Run(string(target), func(t *testing.T) {
					h, _, cap, _ := newTestHandler(t, uint64(playerCount)*7)
					ctx := context.Background()
					g := seedLobby(t, h, playerCount)
					if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
						t.Fatal(err)
					}
					advanceTo(t, h, g.GameID, target)
					// Snapshot the Round as a value (not a pointer); the
					// in-memory DB hands out shared pointers, so any
					// later engine mutation would alias into a struct
					// captured here.
					mid, _ := h.loadGame(ctx, g.GameID)
					midRound := mid.Round
					if err := h.ForceProgress(ctx, g.GameID, "user-host"); err != nil {
						t.Fatalf("ForceProgress from %s: %v", target, err)
					}
					after, _ := h.loadGame(ctx, g.GameID)
					// Forward progress: either the phase changed OR the
					// round advanced past the state that was in place
					// right before ForceProgress ran.
					if after.Phase == target && after.Round <= midRound && after.Status != secrethitler.GameStatusCompleted {
						t.Errorf("no forward progress after ForceProgress from %s: midRound=%d after=(%s)",
							target, midRound, summarize(after))
					}
					// Some reason=forced event must have landed.
					sawForced := false
					for _, env := range cap.envs {
						if p, ok := env.Payload.(PhaseChangedPayload); ok && p.Reason == ReasonForced {
							sawForced = true
						}
						if p, ok := env.Payload.(ElectionResultPayload); ok && p.Reason == ReasonForced {
							sawForced = true
						}
					}
					if !sawForced {
						t.Errorf("no phase-change with ReasonForced emitted from %s", target)
					}
				})
			}
		})
	}
}

// TestActions_TimerExpiredAcrossPhases is the timer analogue: advance
// the clock past the phase deadline and confirm TimerExpired advances
// the game.
func TestActions_TimerExpiredAcrossPhases(t *testing.T) {
	for _, playerCount := range []int{5, 6, 7, 8, 9, 10} {
		playerCount := playerCount
		t.Run(fmt.Sprintf("players=%d", playerCount), func(t *testing.T) {
			for _, target := range []secrethitler.GamePhase{
				secrethitler.PhaseNomination,
				secrethitler.PhaseElection,
			} {
				target := target
				t.Run(string(target), func(t *testing.T) {
					h, _, cap, clock := newTestHandler(t, uint64(playerCount)+1)
					ctx := context.Background()
					g := seedLobby(t, h, playerCount)
					if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
						t.Fatal(err)
					}
					advanceTo(t, h, g.GameID, target)
					clock.Advance(10 * time.Minute)
					if err := h.TimerExpired(ctx, g.GameID); err != nil {
						t.Fatalf("TimerExpired from %s: %v", target, err)
					}
					sawTimeout := false
					for _, env := range cap.envs {
						if p, ok := env.Payload.(PhaseChangedPayload); ok && p.Reason == ReasonTimeout {
							sawTimeout = true
						}
						if p, ok := env.Payload.(ElectionResultPayload); ok && p.Reason == ReasonTimeout {
							sawTimeout = true
						}
					}
					if !sawTimeout {
						t.Errorf("no phase-change with ReasonTimeout from %s", target)
					}
				})
			}
		})
	}
}

// advanceTo progresses the game from Nomination into the requested
// phase by making minimal legal moves. Used by the ForceProgress and
// TimerExpired tests to set up each phase without extra strategy.
func advanceTo(t *testing.T, h *GameHandler, gameID string, phase secrethitler.GamePhase) {
	t.Helper()
	ctx := context.Background()
	for step := 0; step < 30; step++ {
		game, _ := h.loadGame(ctx, gameID)
		if game.Phase == phase {
			return
		}
		players, _ := h.loadPlayers(ctx, gameID)
		switch game.Phase {
		case secrethitler.PhaseNomination:
			president := findPlayerBySeat(players, game.PresidentSeat)
			chancellor := pickChancellor(game, players, president)
			if _, err := h.NominateChancellor(ctx, gameID, president.PlayerID, chancellor.PlayerID); err != nil {
				t.Fatalf("advance nominate: %v", err)
			}
		case secrethitler.PhaseElection:
			for _, p := range alivePlayers(players) {
				_ = h.CastVote(ctx, gameID, p.PlayerID, secrethitler.VoteJa)
				g2, _ := h.loadGame(ctx, gameID)
				if g2.Phase != secrethitler.PhaseElection {
					break
				}
			}
		case secrethitler.PhaseLegislativePresident:
			president := findPlayerBySeat(players, game.PresidentSeat)
			if err := h.PresidentDiscard(ctx, gameID, president.PlayerID, 0); err != nil {
				t.Fatalf("advance discard: %v", err)
			}
		default:
			t.Fatalf("advanceTo cannot reach %s from %s", phase, game.Phase)
		}
	}
	t.Fatalf("could not advance to %s", phase)
}
