package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// seedFiveLobby seats 5 players and returns the handler state. No roles
// yet; game is still in lobby.
func seedFiveLobby(t *testing.T, h *GameHandler) (*models.Game, []*models.Player) {
	t.Helper()
	ctx := context.Background()
	g, host, err := h.CreateGame(ctx, "user-host", "ABCDE", "Host")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	players := []*models.Player{host}
	for i := 2; i <= 5; i++ {
		_, p, err := h.JoinGame(ctx, "ABCDE", uniqueUserID(i), uniqueName(i))
		if err != nil {
			t.Fatalf("JoinGame %d: %v", i, err)
		}
		players = append(players, p)
	}
	return g, players
}

func uniqueUserID(i int) string { return "user-" + string(rune('0'+i)) }
func uniqueName(i int) string   { return "P" + string(rune('0'+i)) }

func TestCreateGame_EmitsCreatedAndJoined(t *testing.T) {
	h, _, cap, _ := newTestHandler(t, 1)

	_, host, err := h.CreateGame(context.Background(), "user-1", "ABCDE", "Alice")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if host.Seat != 0 || !host.IsHost {
		t.Errorf("host seat/isHost wrong: %+v", host)
	}

	types := cap.typesEmitted()
	if len(types) < 2 ||
		types[0] != string(secrethitler.EventGameCreated) ||
		types[1] != string(secrethitler.EventPlayerJoined) {
		t.Errorf("want [game_created, player_joined, ...]; got %v", types)
	}
}

func TestJoinGame_DeduplicatesByUserID(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 1)
	ctx := context.Background()
	_, _, _ = h.CreateGame(ctx, "user-1", "ABCDE", "Host")

	_, p1, err := h.JoinGame(ctx, "ABCDE", "user-2", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	_, p2, err := h.JoinGame(ctx, "ABCDE", "user-2", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	if p1.PlayerID != p2.PlayerID {
		t.Error("rejoining same user should return the existing player")
	}
}

func TestStartGame_RandomPresidentAndRoleDeal(t *testing.T) {
	h, _, cap, _ := newTestHandler(t, 42)
	ctx := context.Background()
	g, players := seedFiveLobby(t, h)

	got, err := h.StartGame(ctx, g.GameID, "user-host")
	if err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if got.Status != secrethitler.GameStatusInProgress {
		t.Errorf("Status = %s", got.Status)
	}
	if got.Phase != secrethitler.PhaseNomination {
		t.Errorf("Phase = %s", got.Phase)
	}
	if got.PresidentSeat < 0 || got.PresidentSeat >= len(players) {
		t.Errorf("PresidentSeat out of range: %d", got.PresidentSeat)
	}
	if got.PlayerCount != 5 {
		t.Errorf("PlayerCount = %d", got.PlayerCount)
	}
	if len(got.DrawPile) != secrethitler.LiberalPoliciesInDeck+secrethitler.FascistPoliciesInDeck {
		t.Errorf("DrawPile size = %d", len(got.DrawPile))
	}

	// Role distribution for 5p: 3 liberal, 1 fascist, 1 hitler.
	fresh, _ := h.loadPlayers(ctx, g.GameID)
	libs, fascs, hits := 0, 0, 0
	for _, p := range fresh {
		switch p.Role {
		case secrethitler.RoleLiberal:
			libs++
		case secrethitler.RoleFascist:
			fascs++
		case secrethitler.RoleHitler:
			hits++
		}
	}
	if libs != 3 || fascs != 1 || hits != 1 {
		t.Errorf("role distribution wrong: lib=%d fasc=%d hitler=%d", libs, fascs, hits)
	}

	// Every player gets a whispered role event.
	roles := cap.envelopesOfType(string(secrethitler.EventRolesAssigned))
	privates := 0
	for _, e := range roles {
		if e.Audience.Scope == AudiencePlayer {
			privates++
		}
	}
	if privates != 5 {
		t.Errorf("expected 5 private role whispers, got %d", privates)
	}
}

func TestStartGame_RejectsNonHost(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 1)
	g, _ := seedFiveLobby(t, h)
	_, err := h.StartGame(context.Background(), g.GameID, "user-2")
	if !errors.Is(err, ErrNotHost) {
		t.Errorf("want ErrNotHost, got %v", err)
	}
}

func TestStartGame_RejectsNotEnoughPlayers(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 1)
	ctx := context.Background()
	g, _, _ := h.CreateGame(ctx, "user-1", "ABCDE", "Host")
	_, err := h.StartGame(ctx, g.GameID, "user-1")
	if !errors.Is(err, ErrNotEnoughPlayers) {
		t.Errorf("want ErrNotEnoughPlayers, got %v", err)
	}
}

func TestCastVote_AllVotedResolvesElection(t *testing.T) {
	h, _, cap, _ := newTestHandler(t, 7)
	ctx := context.Background()
	g, players := seedFiveLobby(t, h)
	started, err := h.StartGame(ctx, g.GameID, "user-host")
	if err != nil {
		t.Fatal(err)
	}

	// Refresh players so we see their seats/ids.
	players, _ = h.loadPlayers(ctx, g.GameID)
	president := findPlayerBySeat(players, started.PresidentSeat)
	chancellor := findFirstOther(players, president.PlayerID)

	if _, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}

	// Everyone votes Ja.
	for _, p := range players {
		if err := h.CastVote(ctx, g.GameID, p.PlayerID, secrethitler.VoteJa); err != nil {
			t.Fatalf("CastVote for %s: %v", p.PlayerID, err)
		}
	}

	// Election should have resolved: ReasonAllVoted.
	results := cap.envelopesOfType(string(secrethitler.EventElectionResult))
	if len(results) == 0 {
		t.Fatal("expected election-result emission")
	}
	found := false
	for _, env := range results {
		if p, ok := env.Payload.(ElectionResultPayload); ok && p.Reason == ReasonAllVoted && p.Passed {
			found = true
		}
	}
	if !found {
		t.Errorf("expected passed ElectionResultPayload with ReasonAllVoted; got %+v", results)
	}

	post, _ := h.loadGame(ctx, g.GameID)
	if post.Phase != secrethitler.PhaseLegislativePresident {
		t.Errorf("post-election phase = %s, want legislative_president", post.Phase)
	}
}

func TestCastVote_RejectsDoubleVote(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 7)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)
	started, _ := h.StartGame(ctx, g.GameID, "user-host")

	players, _ := h.loadPlayers(ctx, g.GameID)
	president := findPlayerBySeat(players, started.PresidentSeat)
	chancellor := findFirstOther(players, president.PlayerID)
	_, _ = h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID)

	if err := h.CastVote(ctx, g.GameID, players[0].PlayerID, secrethitler.VoteJa); err != nil {
		t.Fatal(err)
	}
	err := h.CastVote(ctx, g.GameID, players[0].PlayerID, secrethitler.VoteNein)
	if !errors.Is(err, ErrAlreadyVoted) {
		t.Errorf("want ErrAlreadyVoted, got %v", err)
	}
}

func TestForceProgress_AsHostAdvancesElection(t *testing.T) {
	h, _, cap, _ := newTestHandler(t, 11)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)
	started, _ := h.StartGame(ctx, g.GameID, "user-host")
	players, _ := h.loadPlayers(ctx, g.GameID)
	president := findPlayerBySeat(players, started.PresidentSeat)
	chancellor := findFirstOther(players, president.PlayerID)
	_, _ = h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID)

	// Only two players vote (ja).
	_ = h.CastVote(ctx, g.GameID, players[0].PlayerID, secrethitler.VoteJa)
	_ = h.CastVote(ctx, g.GameID, players[1].PlayerID, secrethitler.VoteJa)

	// Host forces progress while phase is still Election.
	if err := h.ForceProgress(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("ForceProgress: %v", err)
	}

	// Expect an ElectionResult with ReasonForced.
	results := cap.envelopesOfType(string(secrethitler.EventElectionResult))
	found := false
	for _, env := range results {
		if p, ok := env.Payload.(ElectionResultPayload); ok && p.Reason == ReasonForced {
			found = true
		}
	}
	if !found {
		t.Error("expected ElectionResult with ReasonForced")
	}
}

func TestForceProgress_RequiresHost(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 1)
	g, _ := seedFiveLobby(t, h)
	if _, err := h.StartGame(context.Background(), g.GameID, "user-host"); err != nil {
		t.Fatal(err)
	}
	err := h.ForceProgress(context.Background(), g.GameID, "user-2")
	if !errors.Is(err, ErrNotHost) {
		t.Errorf("want ErrNotHost, got %v", err)
	}
}

func TestTimerExpired_AdvancesAfterDeadline(t *testing.T) {
	h, _, cap, clock := newTestHandler(t, 1)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)
	started, _ := h.StartGame(ctx, g.GameID, "user-host")
	players, _ := h.loadPlayers(ctx, g.GameID)
	president := findPlayerBySeat(players, started.PresidentSeat)
	chancellor := findFirstOther(players, president.PlayerID)
	_, _ = h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID)

	// Before deadline: refused.
	err := h.TimerExpired(ctx, g.GameID)
	if !errors.Is(err, ErrDeadlineNotReached) {
		t.Errorf("want ErrDeadlineNotReached, got %v", err)
	}

	// After deadline: election resolves via timeout.
	clock.Advance(10 * time.Minute)
	if err := h.TimerExpired(ctx, g.GameID); err != nil {
		t.Fatalf("TimerExpired: %v", err)
	}
	found := false
	for _, env := range cap.envelopesOfType(string(secrethitler.EventElectionResult)) {
		if p, ok := env.Payload.(ElectionResultPayload); ok && p.Reason == ReasonTimeout {
			found = true
		}
	}
	if !found {
		t.Error("expected ElectionResult with ReasonTimeout")
	}
}

func TestNominateChancellor_RejectsSelfNomination(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 1)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)
	started, _ := h.StartGame(ctx, g.GameID, "user-host")
	players, _ := h.loadPlayers(ctx, g.GameID)
	president := findPlayerBySeat(players, started.PresidentSeat)
	_, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, president.PlayerID)
	if !errors.Is(err, ErrIneligibleCandidate) {
		t.Errorf("want ErrIneligibleCandidate, got %v", err)
	}
}

// findFirstOther returns any player that is not the given ID. Helper
// for tests that need "pick anyone but the president".
func findFirstOther(players []*models.Player, excludeID string) *models.Player {
	for _, p := range players {
		if p.PlayerID != excludeID {
			return p
		}
	}
	return nil
}
