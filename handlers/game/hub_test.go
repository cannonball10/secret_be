package game

import (
	"context"
	"testing"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

func TestMemoryHub_BroadcastReachesAllSubscribers(t *testing.T) {
	h := NewMemoryHub()
	board := h.Subscribe("game-1", DeviceRoleBoard, "")
	p1 := h.Subscribe("game-1", DeviceRolePlayer, "player-1")
	p2 := h.Subscribe("game-1", DeviceRolePlayer, "player-2")

	env := Envelope{
		GameID:   "game-1",
		Event:    models.NewGameEvent("game-1", secrethitler.EventGameStarted, ""),
		Audience: Audience{Scope: AudienceBroadcast},
	}
	h.Publish(context.Background(), env)

	for _, sub := range []*Subscriber{board, p1, p2} {
		select {
		case got := <-sub.Send:
			if got.GameID != "game-1" {
				t.Errorf("sub %s got wrong gameID: %v", sub.ID, got.GameID)
			}
		default:
			t.Errorf("sub %s did not receive broadcast", sub.ID)
		}
	}
}

func TestMemoryHub_WhisperOnlyReachesTarget(t *testing.T) {
	h := NewMemoryHub()
	board := h.Subscribe("game-1", DeviceRoleBoard, "")
	p1 := h.Subscribe("game-1", DeviceRolePlayer, "player-1")
	p2 := h.Subscribe("game-1", DeviceRolePlayer, "player-2")

	env := Envelope{
		GameID:   "game-1",
		Event:    models.NewGameEvent("game-1", secrethitler.EventRolesAssigned, ""),
		Audience: Audience{Scope: AudiencePlayer, PlayerID: "player-1"},
	}
	h.Publish(context.Background(), env)

	// p1 gets it; board and p2 do not.
	select {
	case <-p1.Send:
	default:
		t.Error("player-1 should have received the whisper")
	}
	select {
	case <-board.Send:
		t.Error("board should not receive whispers")
	default:
	}
	select {
	case <-p2.Send:
		t.Error("player-2 should not receive player-1's whisper")
	default:
	}
}

func TestMemoryHub_UnsubscribeClosesChannel(t *testing.T) {
	h := NewMemoryHub()
	sub := h.Subscribe("game-1", DeviceRolePlayer, "player-1")

	h.Unsubscribe("game-1", sub.ID)

	// Channel should now be closed.
	_, ok := <-sub.Send
	if ok {
		t.Error("expected Send channel to be closed after unsubscribe")
	}
	if got := h.SubscriberCount("game-1"); got != 0 {
		t.Errorf("SubscriberCount = %d, want 0", got)
	}
}

func TestMemoryHub_SlowSubscriberDoesNotBlock(t *testing.T) {
	h := NewMemoryHub()
	sub := h.Subscribe("game-1", DeviceRoleBoard, "")

	// Fill the buffered channel without reading.
	env := Envelope{
		GameID:   "game-1",
		Event:    models.NewGameEvent("game-1", secrethitler.EventVoteCast, ""),
		Audience: Audience{Scope: AudienceBroadcast},
	}
	// Publish more than buffer size; should not deadlock.
	for i := 0; i < 100; i++ {
		h.Publish(context.Background(), env)
	}
	// Drain and make sure at least some messages arrived.
	count := 0
drain:
	for {
		select {
		case <-sub.Send:
			count++
		default:
			break drain
		}
	}
	if count == 0 {
		t.Error("expected at least one message to be buffered")
	}
}

func TestHubEmitter_PublishesThroughHub(t *testing.T) {
	h := NewMemoryHub()
	sub := h.Subscribe("game-1", DeviceRoleBoard, "")
	e := NewHubEmitter(h)

	e.Emit(context.Background(), Envelope{
		GameID:   "game-1",
		Event:    models.NewGameEvent("game-1", secrethitler.EventGameEnded, ""),
		Audience: Audience{Scope: AudienceBroadcast},
	})

	select {
	case got := <-sub.Send:
		if got.Event.Type != secrethitler.EventGameEnded {
			t.Errorf("type = %q", got.Event.Type)
		}
	default:
		t.Error("expected envelope through HubEmitter")
	}
}
