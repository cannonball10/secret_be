package game

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
)

// memoryDB is an in-memory DatabaseConnector used for engine tests. It
// supports only the operations the engine actually uses.
type memoryDB struct {
	mu    sync.Mutex
	items map[string]models.Model
}

func newMemoryDB() *memoryDB {
	return &memoryDB{items: make(map[string]models.Model)}
}

func dbKey(pk, sk string) string { return pk + "||" + sk }

func (m *memoryDB) Get(_ context.Context, _ *string, key database.Key) (models.Model, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[dbKey(key["PK"], key["SK"])], nil
}

func (m *memoryDB) Query(_ context.Context, _ *string, input database.QueryInput, _ database.QueryOptions) (*database.QueryOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// GSI query: match against the model's GSI key pair.
	if input.IndexName != nil {
		gsiNum := 0
		switch *input.IndexName {
		case database.IndexName_GSI1:
			gsiNum = 1
		case database.IndexName_GSI2:
			gsiNum = 2
		}
		results := make([]any, 0)
		for _, v := range m.items {
			pair, ok := v.GSIs()[gsiNum]
			if !ok {
				continue
			}
			if pair.PK != input.PartitionKey {
				continue
			}
			if input.SortKey != nil && input.SortKey.BeginsWith != nil {
				if !strings.HasPrefix(pair.SK, *input.SortKey.BeginsWith) {
					continue
				}
			}
			if input.SortKey != nil && input.SortKey.EQ != nil {
				if pair.SK != *input.SortKey.EQ {
					continue
				}
			}
			results = append(results, v)
		}
		return &database.QueryOutput{Models: results}, nil
	}

	// Base-table query: match on PK (and optional SK begins_with).
	results := make([]any, 0)
	for k, v := range m.items {
		parts := strings.SplitN(k, "||", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] != input.PartitionKey {
			continue
		}
		if input.SortKey != nil && input.SortKey.BeginsWith != nil {
			if !strings.HasPrefix(parts[1], *input.SortKey.BeginsWith) {
				continue
			}
		}
		results = append(results, v)
	}
	return &database.QueryOutput{Models: results}, nil
}

func (m *memoryDB) Upsert(_ context.Context, _ *string, item models.Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[dbKey(item.PK(), item.SK())] = item
	return nil
}

func (m *memoryDB) Delete(_ context.Context, _ *string, key database.Key) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, dbKey(key["PK"], key["SK"]))
	return nil
}

func (m *memoryDB) BulkGet(_ context.Context, _ *string, keys []database.Key) ([]models.Model, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Model, 0, len(keys))
	for _, k := range keys {
		if v, ok := m.items[dbKey(k["PK"], k["SK"])]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *memoryDB) BulkUpsert(_ context.Context, _ *string, items []models.Model, _ *database.BulkOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range items {
		m.items[dbKey(item.PK(), item.SK())] = item
	}
	return nil
}

func (m *memoryDB) BulkDelete(_ context.Context, _ *string, keys []database.Key, _ *database.BulkOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.items, dbKey(k["PK"], k["SK"]))
	}
	return nil
}

func (m *memoryDB) ListCollections(context.Context) ([]string, error) { return nil, nil }
func (m *memoryDB) CreateCollection(context.Context, *string) error   { return nil }
func (m *memoryDB) DeleteCollection(context.Context, *string) error   { return nil }

// captureEmitter records every Envelope for later assertion.
type captureEmitter struct {
	mu   sync.Mutex
	envs []Envelope
}

func (c *captureEmitter) Emit(_ context.Context, env Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.envs = append(c.envs, env)
}

// typesEmitted returns the list of event types in order.
func (c *captureEmitter) typesEmitted() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.envs))
	for i, e := range c.envs {
		out[i] = string(e.Event.Type)
	}
	return out
}

// envelopesOfType returns every envelope with the given event type.
func (c *captureEmitter) envelopesOfType(t string) []Envelope {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Envelope, 0)
	for _, e := range c.envs {
		if string(e.Event.Type) == t {
			out = append(out, e)
		}
	}
	return out
}

// newTestHandler builds a GameHandler wired to an in-memory DB, a
// capture emitter, a fake clock, and a deterministic RNG.
func newTestHandler(t *testing.T, seed uint64) (*GameHandler, *memoryDB, *captureEmitter, *FakeClock) {
	t.Helper()
	db := newMemoryDB()
	cap := &captureEmitter{}
	clock := &FakeClock{Current: mustParseTime("2026-04-18T12:00:00Z")}
	rng := NewSeededRNG(seed)
	return NewGameHandler(db,
		WithEmitter(cap),
		WithClock(clock),
		WithRNG(rng),
	), db, cap, clock
}
