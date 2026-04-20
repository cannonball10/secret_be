package game

import (
	"context"
	"sync"
)

// Subscriber is a single device listening for envelopes on a game. It
// is intentionally transport-agnostic: SSE, WebSocket, and Redis
// fan-out all look the same to the Hub.
type Subscriber struct {
	// ID is a locally-unique identifier for this subscription. It is
	// returned by Hub.Subscribe and used by Unsubscribe to remove the
	// subscription when the client disconnects.
	ID string

	// GameID scopes the subscription to a single game.
	GameID string

	// DeviceRole is "board" for the host-owned display or "player" for
	// a player's phone. Board devices only receive broadcast envelopes;
	// player devices receive broadcasts plus whispers targeted at their
	// PlayerID.
	DeviceRole DeviceRole

	// PlayerID, if set, is the player this device acts on behalf of.
	// Required when DeviceRole == DeviceRolePlayer.
	PlayerID string

	// Send is the channel the Hub writes envelopes to. It is buffered;
	// if the subscriber falls behind the Hub drops the message rather
	// than blocking the engine.
	Send chan Envelope
}

// DeviceRole enumerates the kinds of devices that can subscribe.
type DeviceRole string

const (
	// DeviceRoleBoard is the host's "game board" display — it never
	// votes and never receives secret whispers.
	DeviceRoleBoard DeviceRole = "board"
	// DeviceRolePlayer is a player's mobile app. Receives broadcasts
	// plus whispers addressed to its PlayerID.
	DeviceRolePlayer DeviceRole = "player"
)

// Hub fans out engine envelopes to subscribers. It is the only
// stateful piece of the streaming stack; REST handlers open SSE
// connections, register with the Hub, and forward whatever lands on
// their Send channel.
//
// The interface is abstracted so it can be swapped for a Redis pub-sub
// implementation when the service runs multi-process. The in-memory
// MemoryHub provided here is sufficient for single-node deployments.
type Hub interface {
	// Subscribe registers a subscriber and returns it (with a freshly
	// allocated ID and Send channel). The caller owns the channel and
	// must call Unsubscribe when done.
	Subscribe(gameID string, role DeviceRole, playerID string) *Subscriber

	// Unsubscribe removes a subscriber by ID and closes its Send channel.
	Unsubscribe(gameID, subscriberID string)

	// Publish fans an envelope out to every matching subscriber. It
	// never blocks: subscribers whose Send channel is full simply miss
	// the event, which is recoverable since the engine also persists
	// state to the database.
	Publish(ctx context.Context, env Envelope)
}

// MemoryHub is a single-process in-memory Hub suitable for development
// and small deployments. For horizontal scaling, back this with Redis
// pub-sub (same interface).
type MemoryHub struct {
	mu   sync.RWMutex
	subs map[string]map[string]*Subscriber // gameID -> subID -> sub
	seq  uint64
}

// NewMemoryHub returns an initialised in-memory Hub.
func NewMemoryHub() *MemoryHub {
	return &MemoryHub{subs: make(map[string]map[string]*Subscriber)}
}

// subscriberBufferSize is how many envelopes can be queued per
// subscriber before the Hub starts dropping. 256 comfortably absorbs
// a full round's burst (nominate + cable phase + election + vote
// fan-out + legislative + optional executive + narrator audio) with
// headroom for the next round to start queueing before the client
// has drained the prior batch. Below ~64 we saw clients miss the
// chancellor_nominated event when the simulator stacked rounds
// faster than React could render.
const subscriberBufferSize = 256

// Subscribe implements Hub.
func (h *MemoryHub) Subscribe(gameID string, role DeviceRole, playerID string) *Subscriber {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	sub := &Subscriber{
		ID:         subID(h.seq),
		GameID:     gameID,
		DeviceRole: role,
		PlayerID:   playerID,
		Send:       make(chan Envelope, subscriberBufferSize),
	}
	game, ok := h.subs[gameID]
	if !ok {
		game = make(map[string]*Subscriber)
		h.subs[gameID] = game
	}
	game[sub.ID] = sub
	return sub
}

// Unsubscribe implements Hub.
func (h *MemoryHub) Unsubscribe(gameID, subscriberID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if game, ok := h.subs[gameID]; ok {
		if sub, ok := game[subscriberID]; ok {
			close(sub.Send)
			delete(game, subscriberID)
		}
		if len(game) == 0 {
			delete(h.subs, gameID)
		}
	}
}

// Publish implements Hub. Envelopes are dispatched according to their
// Audience scope:
//   - AudienceBroadcast: every subscriber in the game.
//   - AudiencePlayer   : only player-role subscribers whose PlayerID
//     matches the envelope's target.
func (h *MemoryHub) Publish(_ context.Context, env Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	game, ok := h.subs[env.GameID]
	if !ok {
		return
	}
	for _, sub := range game {
		if !shouldDeliver(sub, env) {
			continue
		}
		// Non-blocking send: drop if the subscriber is slow.
		select {
		case sub.Send <- env:
		default:
		}
	}
}

// SubscriberCount returns how many active subscribers a game has
// (useful for health checks and tests).
func (h *MemoryHub) SubscriberCount(gameID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[gameID])
}

// shouldDeliver returns true if the subscriber should receive the
// given envelope.
func shouldDeliver(sub *Subscriber, env Envelope) bool {
	switch env.Audience.Scope {
	case AudienceBroadcast:
		return true
	case AudiencePlayer:
		return sub.DeviceRole == DeviceRolePlayer && sub.PlayerID == env.Audience.PlayerID
	}
	return false
}

// subID formats a monotonic counter as a subscription ID.
func subID(n uint64) string {
	// Small, human-readable id: "sub-<n>".
	const digits = "0123456789abcdef"
	buf := make([]byte, 0, 16)
	buf = append(buf, "sub-"...)
	if n == 0 {
		return string(append(buf, '0'))
	}
	var rev [16]byte
	i := 0
	for n > 0 {
		rev[i] = digits[n&0xf]
		n >>= 4
		i++
	}
	for j := i - 1; j >= 0; j-- {
		buf = append(buf, rev[j])
	}
	return string(buf)
}

// HubEmitter adapts a Hub to the Emitter interface so the engine can
// push envelopes straight into the pub-sub layer. This is the glue
// between the state machine and the network.
type HubEmitter struct {
	Hub Hub
}

// NewHubEmitter returns an Emitter backed by the given Hub.
func NewHubEmitter(h Hub) HubEmitter { return HubEmitter{Hub: h} }

// Emit implements Emitter.
func (e HubEmitter) Emit(ctx context.Context, env Envelope) {
	if e.Hub == nil {
		return
	}
	e.Hub.Publish(ctx, env)
}
