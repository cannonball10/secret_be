package game

import (
	"context"
	"fmt"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/replicant"
)

// CastVote records a player's ja/nein on the active government. If this
// was the last outstanding alive player's vote the engine immediately
// resolves the election (ReasonAllVoted); otherwise we just emit a
// VoteCast ping and wait for the remaining voters or the timer.
func (h *GameHandler) CastVote(ctx context.Context, gameID, playerID string, choice replicant.VoteChoice) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if err := mustPhase(game, replicant.PhaseElection); err != nil {
		return err
	}

	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	voter := findPlayerByID(players, playerID)
	if voter == nil {
		return ErrPlayerNotFound
	}
	if !voter.IsAlive {
		return fmt.Errorf("%w: dead players cannot vote", ErrIneligibleCandidate)
	}

	existing, err := h.loadVotesForGovernment(ctx, gameID, game.CurrentGovernmentID)
	if err != nil {
		return err
	}
	for _, v := range existing {
		if v.PlayerID == playerID {
			return ErrAlreadyVoted
		}
	}

	vote := models.NewVote(gameID, game.CurrentGovernmentID, playerID, choice)
	if err := h.db.Upsert(ctx, nil, vote); err != nil {
		return err
	}

	ev := models.NewGameEvent(gameID, replicant.EventVoteCast, playerID)
	h.broadcast(ctx, ev, VoteCastPayload{
		GovernmentID: game.CurrentGovernmentID,
		PlayerID:     playerID,
	})

	existing = append(existing, vote)
	if haveAllVoted(players, existing) {
		_, _, err := h.resolveElection(ctx, game, players, existing, ReasonAllVoted)
		return err
	}
	return nil
}

// haveAllVoted returns true iff every alive player has a vote record.
func haveAllVoted(players []*models.Player, votes []*models.Vote) bool {
	voted := make(map[string]bool, len(votes))
	for _, v := range votes {
		voted[v.PlayerID] = true
	}
	for _, p := range players {
		if p.IsAlive && !voted[p.PlayerID] {
			return false
		}
	}
	return true
}

// resolveElection tallies the votes on the current government, updates
// the Government record, and advances the phase. The `reason` argument
// is attached to the election-result event so clients know whether the
// resolution was driven by unanimous voting, the timer, or a host force.
//
// Returns the resolved Government, whether it passed, and any error.
// Missing votes (timer/force) count as Nein.
func (h *GameHandler) resolveElection(ctx context.Context, game *models.Game, players []*models.Player, votes []*models.Vote, reason ProgressReason) (*models.Government, bool, error) {
	gov, err := h.loadGovernment(ctx, game.GameID, game.CurrentGovernmentID)
	if err != nil {
		return nil, false, err
	}

	voteMap := make(map[string]replicant.VoteChoice, len(votes))
	ja, nein := 0, 0
	for _, v := range votes {
		voteMap[v.PlayerID] = v.Choice
		if v.IsJa() {
			ja++
		} else {
			nein++
		}
	}
	// Treat missing votes from alive players as Nein when resolving via
	// timeout or forced progression.
	for _, p := range players {
		if !p.IsAlive {
			continue
		}
		if _, ok := voteMap[p.PlayerID]; !ok {
			voteMap[p.PlayerID] = replicant.VoteNein
			nein++
		}
	}

	gov.RecordElection(ja, nein)
	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return nil, false, err
	}

	passed := gov.Status == replicant.GovernmentStatusPassed
	resultEvent := models.NewGameEvent(game.GameID, replicant.EventElectionResult, "")
	h.broadcast(ctx, resultEvent, ElectionResultPayload{
		GovernmentID: gov.GovernmentID,
		Passed:       passed,
		JaVotes:      ja,
		NeinVotes:    nein,
		Votes:        voteMap,
		Reason:       reason,
	})

	if passed {
		return h.onElectionPassed(ctx, game, gov, players)
	}
	return h.onElectionFailed(ctx, game, players, reason)
}

// onElectionPassed handles the "government elected" path: check Hitler
// chancellor win, else deal policies to the president and transition.
//
// Note: the election tracker is NOT reset here. Per the rulebook the
// tracker only resets when a government actually enacts a policy — an
// elected-then-vetoed agenda leaves the tracker intact. The reset
// therefore lives in applyEnactedPolicy.
func (h *GameHandler) onElectionPassed(ctx context.Context, game *models.Game, gov *models.Government, players []*models.Player) (*models.Government, bool, error) {
	// Codes-transferred chancellor win conditions. Once the AI cabal
	// has pushed the Planetary Committee past the codes-transfer
	// threshold, electing a compromised Chancellor hands them the
	// nuclear codes. Two mutually exclusive winners are possible:
	//
	//   1. Chancellor is the Prime (rogue) → AI cabal victory.
	//   2. Chancellor is the Singularity → kingmaker victory
	//      (Singularity alone wins, stealing the endgame from both
	//       sides). Only reachable when Rules.EnableSingularity is on.
	chancellor := findPlayerBySeat(players, *gov.ChancellorSeat)
	if chancellor != nil && game.RogueZoneActive() {
		if chancellor.IsRogue() {
			return gov, true, h.endGame(ctx, game, replicant.PartyAI, replicant.WinRogueElected)
		}
		if chancellor.IsSingularity() {
			return gov, true, h.endGame(ctx, game, replicant.PartySingularity, replicant.WinSingularityKingmaker)
		}
	}

	// Draw 3 policies for the president.
	drawn := h.drawThree(ctx, game)
	gov.DrawnPolicies = drawn
	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return nil, true, err
	}

	// Whisper policies to the president only.
	president := findPlayerBySeat(players, gov.PresidentSeat)
	if president != nil {
		drawnEvent := models.NewGameEvent(game.GameID, replicant.EventPoliciesDrawn, president.PlayerID)
		h.whisper(ctx, drawnEvent, president.PlayerID, PoliciesDrawnPayload{
			GovernmentID: gov.GovernmentID,
			Policies:     drawn,
		})
	}

	h.setPhase(ctx, game, replicant.PhaseLegislativePresident, ReasonAction)
	if err := h.saveGame(ctx, game); err != nil {
		return nil, true, err
	}
	return gov, true, nil
}

// onElectionFailed advances the election tracker (top-decking at 3),
// rotates president, and returns to Nomination.
func (h *GameHandler) onElectionFailed(ctx context.Context, game *models.Game, players []*models.Player, reason ProgressReason) (*models.Government, bool, error) {
	game.ElectionTracker++
	ev := models.NewGameEvent(game.GameID, replicant.EventElectionTracker, "")
	h.broadcast(ctx, ev, ElectionTrackerPayload{Tracker: game.ElectionTracker})

	if game.ElectionTracker >= game.Rules.ElectionTrackerLimit {
		// Top-deck: enact the top policy, clear term limits, reset tracker.
		// topDeck → applyEnactedPolicy already rotates the presidency
		// and sets the next phase, so we return here without another
		// rotation.
		game.ElectionTracker = 0
		game.PreviousPresidentSeat = nil
		game.PreviousChancellorSeat = nil
		if err := h.topDeck(ctx, game); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}

	h.rotatePresident(game, players)
	h.setPhase(ctx, game, replicant.PhaseNomination, reason)
	if err := h.saveGame(ctx, game); err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

// rotatePresident advances the presidency to the next alive seat. If a
// special election set a return seat, respect it.
func (h *GameHandler) rotatePresident(game *models.Game, players []*models.Player) {
	if game.SpecialElectionReturnSeat != nil {
		game.PresidentSeat = *game.SpecialElectionReturnSeat
		game.SpecialElectionReturnSeat = nil
	} else {
		game.PresidentSeat = nextAliveSeat(players, game.PresidentSeat)
	}
	game.Round++
	game.ChancellorSeat = nil
	game.CurrentGovernmentID = ""
}
