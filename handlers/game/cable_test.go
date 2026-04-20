package game

import (
	"context"
	"testing"
	"time"

	"github.com/cannonball10/foundation/handlers/narrator"
	"github.com/cannonball10/foundation/models"
	audioschema "github.com/cannonball10/foundation/schemas/audio"
	"github.com/cannonball10/foundation/schemas/replicant"
)

// fakeNarrator is a deterministic CableNarrator for tests. RankCables
// returns the score table in nameToScore keyed on cable Body; Speak
// records the cue and returns a canned result.
type fakeNarrator struct {
	rankByBody map[string]float64
	rankErr    error
	speakErr   error
	// Captured by Speak — the last cue the engine asked the narrator
	// to voice. Lets tests assert "silence" vs "leak" without sniffing
	// SSE envelopes.
	lastCue narrator.CueKind
	calls   int
}

func (f *fakeNarrator) RankCables(_ context.Context, cables []narrator.Cable) ([]narrator.ScoredCable, error) {
	if f.rankErr != nil {
		return nil, f.rankErr
	}
	out := make([]narrator.ScoredCable, len(cables))
	for i, c := range cables {
		out[i] = narrator.ScoredCable{MessageID: c.MessageID, Score: f.rankByBody[c.Body], Reason: "fake"}
	}
	return out, nil
}

func (f *fakeNarrator) Speak(_ context.Context, cue narrator.Cue) (*narrator.Result, error) {
	f.calls++
	f.lastCue = cue.Kind
	if f.speakErr != nil {
		return nil, f.speakErr
	}
	return &narrator.Result{
		CueID:  "cue-test",
		Script: "Test broadcast.",
		Audio:  []byte{0x00, 0x01, 0x02},
		Format: audioschema.FormatMP3,
		TookMS: 1,
	}, nil
}

// TestCablePhase_DisabledDefault confirms that with the default rules
// (CableModeDisabled) the nomination → election transition is
// unchanged — no Cable Phase is interposed.
func TestCablePhase_DisabledDefault(t *testing.T) {
	ctx := context.Background()
	h, _, _, _ := newTestHandler(t, 101)
	g := seedLobby(t, h, 7)
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)

	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}

	after, _ := h.loadGame(ctx, g.GameID)
	if after.Phase != replicant.PhaseElection {
		t.Errorf("expected PhaseElection after nomination with cable disabled, got %s", after.Phase)
	}
}

// TestCablePhase_EveryRound_Opens verifies that when CableModeEveryRound
// is on, NominateChancellor transitions to PhaseCablePhase (not
// PhaseElection) and emits the cable_phase_opened event with a
// deadline derived from CablePhaseDurationSec.
func TestCablePhase_EveryRound_Opens(t *testing.T) {
	ctx := context.Background()
	h, _, cap, clock := newTestHandler(t, 103)
	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 30
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)

	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}

	after, _ := h.loadGame(ctx, g.GameID)
	if after.Phase != replicant.PhaseCablePhase {
		t.Errorf("expected PhaseCablePhase after nomination with cable on, got %s", after.Phase)
	}
	if after.PhaseDeadline == nil {
		t.Fatal("expected phase deadline set")
	}
	expectedDeadline := clock.Now().Add(30 * time.Second)
	if !after.PhaseDeadline.Equal(expectedDeadline) {
		t.Errorf("deadline = %v, want %v", after.PhaseDeadline, expectedDeadline)
	}
	if len(cap.envelopesOfType(string(replicant.EventCablePhaseOpened))) != 1 {
		t.Errorf("expected 1 cable_phase_opened event, got %d",
			len(cap.envelopesOfType(string(replicant.EventCablePhaseOpened))))
	}
}

// TestCablePhase_TimerExpired_Closes verifies that letting the cable
// deadline elapse moves the game into election and emits a
// cable_phase_closed event with ReasonTimeout.
func TestCablePhase_TimerExpired_Closes(t *testing.T) {
	ctx := context.Background()
	h, _, cap, clock := newTestHandler(t, 107)
	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 15
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)
	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}

	// Advance the fake clock past the cable deadline, then tick the
	// timer. The engine must advance to election.
	clock.Advance(20 * time.Second)
	if err := h.TimerExpired(ctx, g.GameID); err != nil {
		t.Fatalf("TimerExpired: %v", err)
	}

	after, _ := h.loadGame(ctx, g.GameID)
	if after.Phase != replicant.PhaseElection {
		t.Errorf("expected PhaseElection after cable timeout, got %s", after.Phase)
	}
	closedEvents := cap.envelopesOfType(string(replicant.EventCablePhaseClosed))
	if len(closedEvents) != 1 {
		t.Fatalf("expected 1 cable_phase_closed event, got %d", len(closedEvents))
	}
}

// TestCablePhase_Submission_PersistsAndAcks verifies that SendChat with
// ChannelCable during the Cable Phase persists the message under the
// current governmentId and whispers an Ack=true payload back to the
// sender only (no fan-out). Rejects sends outside the phase.
func TestCablePhase_Submission_PersistsAndAcks(t *testing.T) {
	ctx := context.Background()
	h, db, cap, _ := newTestHandler(t, 131)
	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 60
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)

	// Before cable phase opens, sends are rejected.
	if err := h.SendChat(ctx, g.GameID, players[0].PlayerID, ChannelCable, "early bird"); err == nil {
		t.Error("expected ChannelCable outside Cable Phase to fail")
	}

	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}

	// Cable Phase is now open. A submission should persist + ack.
	sender := players[0]
	before := len(cap.envelopesOfType(string(replicant.EventChatMessage)))
	if err := h.SendChat(ctx, g.GameID, sender.PlayerID, ChannelCable, "I move that we adjourn."); err != nil {
		t.Fatalf("SendChat(cable): %v", err)
	}
	after := cap.envelopesOfType(string(replicant.EventChatMessage))
	if len(after)-before != 1 {
		t.Fatalf("expected 1 new chat_message envelope, got %d", len(after)-before)
	}

	newest := after[len(after)-1]
	if newest.Audience.Scope != AudiencePlayer || newest.Audience.PlayerID != sender.PlayerID {
		t.Errorf("cable ack should whisper to sender; got scope=%q player=%q",
			newest.Audience.Scope, newest.Audience.PlayerID)
	}

	// Confirm the ChatMessage was persisted under the current
	// governmentId. Query via the memoryDB directly.
	gameAfter, _ := h.loadGame(ctx, g.GameID)
	var found int
	for _, item := range db.items {
		m, ok := item.(*models.ChatMessage)
		if !ok {
			continue
		}
		if m.GovernmentID == gameAfter.CurrentGovernmentID && m.Channel == string(ChannelCable) {
			found++
		}
	}
	if found != 1 {
		t.Errorf("expected 1 persisted cable for government %q, got %d",
			gameAfter.CurrentGovernmentID, found)
	}
}

// TestCablePhase_LeakFlow_PicksTopCable wires a fake narrator, submits
// two cables of differing scores, and asserts that the high-scored
// cable is leaked: cable_leaked envelope fires with its body,
// narrator_speak fires with a CueCableLeak script, and the persisted
// ChatMessage has Leaked=true + SubversionScore set.
func TestCablePhase_LeakFlow_PicksTopCable(t *testing.T) {
	ctx := context.Background()
	f := &fakeNarrator{
		rankByBody: map[string]float64{
			"boring procedural chatter":   1,
			"vote nein on Jordan this round": 9,
		},
	}
	db := newMemoryDB()
	cap := &captureEmitter{}
	clock := &FakeClock{Current: mustParseTime("2026-04-18T12:00:00Z")}
	h := NewGameHandler(db,
		WithEmitter(cap),
		WithClock(clock),
		WithRNG(NewSeededRNG(211)),
		WithCableNarrator(f),
	)

	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 60
	g.Rules.CableLeakSilenceChance = 0 // force a leak
	// This test asserts the author round-trips into the leaked payload,
	// which the anonymous-leak default would strip. Disable it.
	g.Rules.AnonymousCableLeaks = false
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)
	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}

	// Two players submit cables during the phase.
	lowAuthor := players[0]
	highAuthor := players[1]
	if err := h.SendChat(ctx, g.GameID, lowAuthor.PlayerID, ChannelCable, "boring procedural chatter"); err != nil {
		t.Fatalf("low cable: %v", err)
	}
	if err := h.SendChat(ctx, g.GameID, highAuthor.PlayerID, ChannelCable, "vote nein on Jordan this round"); err != nil {
		t.Fatalf("high cable: %v", err)
	}

	// Timer expiry triggers the leak flow.
	clock.Advance(65 * time.Second)
	if err := h.TimerExpired(ctx, g.GameID); err != nil {
		t.Fatalf("TimerExpired: %v", err)
	}

	// Exactly one cable_leaked envelope, naming the high-scored body.
	leaks := cap.envelopesOfType(string(replicant.EventCableLeaked))
	if len(leaks) != 1 {
		t.Fatalf("expected 1 cable_leaked, got %d", len(leaks))
	}
	payload, ok := leaks[0].Payload.(CableLeakedPayload)
	if !ok {
		t.Fatalf("cable_leaked payload wrong type: %T", leaks[0].Payload)
	}
	if payload.Silenced {
		t.Error("expected a leak, got silence")
	}
	if payload.Body != "vote nein on Jordan this round" {
		t.Errorf("leaked body = %q, want the high-scored cable", payload.Body)
	}
	if payload.AuthorPlayerID != highAuthor.PlayerID {
		t.Errorf("leaked authorId = %q, want %q", payload.AuthorPlayerID, highAuthor.PlayerID)
	}
	if payload.SubversionScore != 9 {
		t.Errorf("leaked score = %v, want 9", payload.SubversionScore)
	}

	// narrator_speak emitted with CueCableLeak.
	if f.lastCue != narrator.CueCableLeak {
		t.Errorf("narrator last cue = %q, want %q", f.lastCue, narrator.CueCableLeak)
	}
	if f.calls != 1 {
		t.Errorf("expected narrator called once, got %d", f.calls)
	}

	// Persisted cable should be marked Leaked.
	for _, item := range db.items {
		if m, ok := item.(*models.ChatMessage); ok && m.Leaked {
			if m.MessageID != payload.MessageID {
				t.Errorf("leaked persist id mismatch: got %q, want %q", m.MessageID, payload.MessageID)
			}
			if m.SubversionScore != 9 {
				t.Errorf("persisted score = %v, want 9", m.SubversionScore)
			}
			return
		}
	}
	t.Error("no ChatMessage row marked Leaked=true")
}

// TestCablePhase_LeakFlow_SilenceFallback verifies that when the RNG
// roll lands below CableLeakSilenceChance, the narrator speaks the
// silence cue and the cable_leaked payload carries Silenced=true with
// an empty body — no cables get marked leaked in the DB.
func TestCablePhase_LeakFlow_SilenceFallback(t *testing.T) {
	ctx := context.Background()
	f := &fakeNarrator{rankByBody: map[string]float64{"anything": 5}}
	db := newMemoryDB()
	cap := &captureEmitter{}
	clock := &FakeClock{Current: mustParseTime("2026-04-18T12:00:00Z")}
	h := NewGameHandler(db,
		WithEmitter(cap),
		WithClock(clock),
		WithRNG(NewSeededRNG(311)),
		WithCableNarrator(f),
	)

	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 60
	g.Rules.CableLeakSilenceChance = 1.0 // force silence
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)
	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}
	if err := h.SendChat(ctx, g.GameID, players[0].PlayerID, ChannelCable, "anything"); err != nil {
		t.Fatalf("SendChat: %v", err)
	}
	clock.Advance(65 * time.Second)
	if err := h.TimerExpired(ctx, g.GameID); err != nil {
		t.Fatalf("TimerExpired: %v", err)
	}

	leaks := cap.envelopesOfType(string(replicant.EventCableLeaked))
	if len(leaks) != 1 {
		t.Fatalf("expected 1 cable_leaked on silence path, got %d", len(leaks))
	}
	payload, ok := leaks[0].Payload.(CableLeakedPayload)
	if !ok {
		t.Fatalf("cable_leaked payload wrong type: %T", leaks[0].Payload)
	}
	if !payload.Silenced {
		t.Error("expected Silenced=true on silence path")
	}
	if payload.Body != "" || payload.MessageID != "" {
		t.Errorf("silence payload should have empty body + id, got body=%q id=%q", payload.Body, payload.MessageID)
	}
	if f.lastCue != narrator.CueCableSilence {
		t.Errorf("narrator last cue = %q, want %q", f.lastCue, narrator.CueCableSilence)
	}
	// No ChatMessage should have Leaked=true.
	for _, item := range db.items {
		if m, ok := item.(*models.ChatMessage); ok && m.Leaked {
			t.Errorf("unexpected Leaked=true on %q in silence path", m.MessageID)
		}
	}
}

// TestCablePhase_LeakFlow_NoSubmissions verifies that when no cables
// were sent during the phase, the engine skips the leak flow entirely
// — no cable_leaked event, no narrator call.
func TestCablePhase_LeakFlow_NoSubmissions(t *testing.T) {
	ctx := context.Background()
	f := &fakeNarrator{}
	db := newMemoryDB()
	cap := &captureEmitter{}
	clock := &FakeClock{Current: mustParseTime("2026-04-18T12:00:00Z")}
	h := NewGameHandler(db,
		WithEmitter(cap),
		WithClock(clock),
		WithRNG(NewSeededRNG(411)),
		WithCableNarrator(f),
	)
	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 60
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)
	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}
	clock.Advance(65 * time.Second)
	if err := h.TimerExpired(ctx, g.GameID); err != nil {
		t.Fatalf("TimerExpired: %v", err)
	}
	if got := len(cap.envelopesOfType(string(replicant.EventCableLeaked))); got != 0 {
		t.Errorf("expected 0 cable_leaked events with no submissions, got %d", got)
	}
	if f.calls != 0 {
		t.Errorf("expected narrator not called with no submissions, got %d", f.calls)
	}
}

// TestCablePhase_ForceProgress_Closes verifies the host can short-circuit
// the cable phase via ForceProgress, landing in election with reason
// "forced".
func TestCablePhase_ForceProgress_Closes(t *testing.T) {
	ctx := context.Background()
	h, _, _, _ := newTestHandler(t, 109)
	g := seedLobby(t, h, 7)
	g.Rules.CablePhaseMode = replicant.CableModeEveryRound
	g.Rules.CablePhaseDurationSec = 60
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	game, _ := h.loadGame(ctx, g.GameID)
	players, _ := h.loadPlayers(ctx, g.GameID)
	pres := findPlayerBySeat(players, game.PresidentSeat)
	chan_ := pickChancellor(game, players, pres)
	if _, err := h.NominateChancellor(ctx, g.GameID, pres.PlayerID, chan_.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}
	if err := h.ForceProgress(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("ForceProgress: %v", err)
	}
	after, _ := h.loadGame(ctx, g.GameID)
	if after.Phase != replicant.PhaseElection {
		t.Errorf("expected PhaseElection after force-progress, got %s", after.Phase)
	}
}
