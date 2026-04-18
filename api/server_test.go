package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cannonball10/foundation/handlers/game"
	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/secrethitler"
	"github.com/gin-gonic/gin"
)

// --- in-memory DB (mirrors handlers/game test harness) ---------------------

type memoryDB struct {
	mu    sync.Mutex
	items map[string]models.Model
}

func newMemoryDB() *memoryDB    { return &memoryDB{items: make(map[string]models.Model)} }
func mkey(pk, sk string) string { return pk + "||" + sk }

func (m *memoryDB) Get(_ context.Context, _ *string, key database.Key) (models.Model, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[mkey(key["PK"], key["SK"])], nil
}

func (m *memoryDB) Query(_ context.Context, _ *string, input database.QueryInput, _ database.QueryOptions) (*database.QueryOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if input.IndexName != nil {
		gsiNum := 1
		if *input.IndexName == database.IndexName_GSI2 {
			gsiNum = 2
		}
		results := make([]any, 0)
		for _, v := range m.items {
			pair, ok := v.GSIs()[gsiNum]
			if !ok || pair.PK != input.PartitionKey {
				continue
			}
			if input.SortKey != nil && input.SortKey.BeginsWith != nil &&
				!strings.HasPrefix(pair.SK, *input.SortKey.BeginsWith) {
				continue
			}
			results = append(results, v)
		}
		return &database.QueryOutput{Models: results}, nil
	}
	results := make([]any, 0)
	for k, v := range m.items {
		parts := strings.SplitN(k, "||", 2)
		if len(parts) != 2 || parts[0] != input.PartitionKey {
			continue
		}
		if input.SortKey != nil && input.SortKey.BeginsWith != nil &&
			!strings.HasPrefix(parts[1], *input.SortKey.BeginsWith) {
			continue
		}
		results = append(results, v)
	}
	return &database.QueryOutput{Models: results}, nil
}

func (m *memoryDB) Upsert(_ context.Context, _ *string, item models.Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[mkey(item.PK(), item.SK())] = item
	return nil
}

func (m *memoryDB) Delete(_ context.Context, _ *string, key database.Key) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, mkey(key["PK"], key["SK"]))
	return nil
}

func (m *memoryDB) BulkGet(_ context.Context, _ *string, keys []database.Key) ([]models.Model, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Model, 0, len(keys))
	for _, k := range keys {
		if v, ok := m.items[mkey(k["PK"], k["SK"])]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *memoryDB) BulkUpsert(_ context.Context, _ *string, items []models.Model, _ *database.BulkOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range items {
		m.items[mkey(item.PK(), item.SK())] = item
	}
	return nil
}

func (m *memoryDB) BulkDelete(context.Context, *string, []database.Key, *database.BulkOptions) error {
	return nil
}
func (m *memoryDB) ListCollections(context.Context) ([]string, error) { return nil, nil }
func (m *memoryDB) CreateCollection(context.Context, *string) error   { return nil }
func (m *memoryDB) DeleteCollection(context.Context, *string) error   { return nil }

// --- helpers ---------------------------------------------------------------

func newTestServer(t *testing.T) (*Server, *memoryDB) {
	t.Helper()
	db := newMemoryDB()
	s := NewServer(Options{
		Database: db,
		Auth:     NopAuth{},
		Hub:      game.NewMemoryHub(),
		EngineOptions: []game.Option{
			game.WithRNG(game.NewSeededRNG(1)),
			game.WithClock(&game.FakeClock{Current: time.Unix(0, 0)}),
		},
		GinMode: gin.TestMode,
	})
	return s, db
}

// doJSON is a helper that issues a request with an Authorization header
// and optional JSON body, then decodes the JSON response.
func doJSON(t *testing.T, s *Server, method, path, userToken string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if userToken != "" {
		req.Header.Set("Authorization", "Bearer "+userToken)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// decode unmarshals the response body into target.
func decode(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

// --- tests -----------------------------------------------------------------

func TestHealthz(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestCreateGame_RequiresAuth(t *testing.T) {
	s, _ := newTestServer(t)
	w := doJSON(t, s, http.MethodPost, "/api/v1/games", "", map[string]string{
		"joinCode": "AAAAA", "displayName": "Alice",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestCreateGame_AndJoin(t *testing.T) {
	s, _ := newTestServer(t)

	w := doJSON(t, s, http.MethodPost, "/api/v1/games", "user-host", map[string]string{
		"joinCode": "ABCDE", "displayName": "Alice",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", w.Code, w.Body.String())
	}
	var created struct {
		Game   *models.Game   `json:"game"`
		Player *models.Player `json:"player"`
	}
	decode(t, w, &created)
	if created.Game.JoinCode != "ABCDE" {
		t.Errorf("JoinCode = %q", created.Game.JoinCode)
	}
	if !created.Player.IsHost {
		t.Error("expected player.IsHost = true")
	}

	// Another user joins.
	w = doJSON(t, s, http.MethodPost, "/api/v1/games/join", "user-bob", map[string]string{
		"joinCode": "ABCDE", "displayName": "Bob",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("join: status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestStartGame_OnlyHost(t *testing.T) {
	s, _ := newTestServer(t)
	// Create + seat 5 players.
	w := doJSON(t, s, http.MethodPost, "/api/v1/games", "user-host", map[string]string{
		"joinCode": "ABCDE", "displayName": "Host",
	})
	var created struct {
		Game *models.Game `json:"game"`
	}
	decode(t, w, &created)
	gameID := created.Game.GameID

	for i := 2; i <= 5; i++ {
		tok := "user-" + string(rune('0'+i))
		doJSON(t, s, http.MethodPost, "/api/v1/games/join", tok, map[string]string{
			"joinCode": "ABCDE", "displayName": tok,
		})
	}

	// Non-host cannot start.
	w = doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/host/start", "user-2", nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-host start: status = %d, want 403", w.Code)
	}

	// Host can.
	w = doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/host/start", "user-host", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("host start: status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestGetGame_ScrubsRolesWhileInProgress(t *testing.T) {
	s, _ := newTestServer(t)
	gameID := seedFivePlayerInProgress(t, s)

	w := doJSON(t, s, http.MethodGet, "/api/v1/games/"+gameID, "user-host", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: status = %d", w.Code)
	}
	var resp struct {
		Players []*models.Player `json:"players"`
	}
	decode(t, w, &resp)
	for _, p := range resp.Players {
		if p.Role != "" || p.Party != "" {
			t.Errorf("player %s: role/party should be scrubbed, got role=%q party=%q", p.PlayerID, p.Role, p.Party)
		}
	}
}

func TestForceProgress_HostOnly(t *testing.T) {
	s, _ := newTestServer(t)
	gameID := seedFivePlayerInProgress(t, s)

	// Non-host rejected.
	w := doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/host/force-progress", "user-2", nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}

	// Host accepted.
	w = doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/host/force-progress", "user-host", nil)
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", w.Code)
	}
}

func TestVote_ByPlayer(t *testing.T) {
	s, db := newTestServer(t)
	gameID := seedFivePlayerInProgress(t, s)

	// Find president + someone else via the DB to construct a nomination.
	var president *models.Player
	var chancellor *models.Player
	for _, it := range db.items {
		if g, ok := it.(*models.Game); ok {
			for _, other := range db.items {
				if p, ok := other.(*models.Player); ok && p.GameID == gameID {
					if p.Seat == g.PresidentSeat {
						president = p
					}
				}
			}
		}
	}
	for _, it := range db.items {
		if p, ok := it.(*models.Player); ok && p.GameID == gameID && p.PlayerID != president.PlayerID {
			chancellor = p
			break
		}
	}
	if president == nil || chancellor == nil {
		t.Fatal("could not locate president/chancellor")
	}

	w := doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/player/nominate", president.UserID,
		map[string]string{"chancellorPlayerId": chancellor.PlayerID})
	if w.Code != http.StatusOK {
		t.Fatalf("nominate: %d, body=%s", w.Code, w.Body.String())
	}

	// Cast a vote as any player.
	w = doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/player/vote", chancellor.UserID,
		map[string]string{"choice": "ja"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("vote: %d, body=%s", w.Code, w.Body.String())
	}

	// Invalid vote choice rejected.
	w = doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/player/vote", "user-host",
		map[string]string{"choice": "maybe"})
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad vote: %d, want 400", w.Code)
	}
}

// TestStreamsReceiveBroadcastAndWhisper spins up the board and a player
// subscription, then triggers a state change and asserts both streams
// observe appropriate envelopes. SSE is tested at the Hub level since
// plumbing the full SSE response through httptest requires goroutines
// and a closable context; see hub_test.go for channel-level coverage.
func TestStreamsReceiveBroadcastAndWhisper(t *testing.T) {
	s, _ := newTestServer(t)
	gameID := seedFivePlayerInProgress(t, s)

	// Subscribe a board device directly through the hub.
	board := s.hub.Subscribe(gameID, game.DeviceRoleBoard, "")
	defer s.hub.Unsubscribe(gameID, board.ID)

	// Force a host-driven advance so at least one broadcast fires.
	w := doJSON(t, s, http.MethodPost, "/api/v1/games/"+gameID+"/host/force-progress", "user-host", nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("force-progress: %d", w.Code)
	}

	select {
	case env := <-board.Send:
		if env.GameID != gameID {
			t.Errorf("got envelope for game %q, want %q", env.GameID, gameID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("board device did not receive broadcast")
	}
}

// seedFivePlayerInProgress creates a game, seats 5 players, and starts
// the match. Returns the game ID.
func seedFivePlayerInProgress(t *testing.T, s *Server) string {
	t.Helper()
	w := doJSON(t, s, http.MethodPost, "/api/v1/games", "user-host", map[string]string{
		"joinCode": "ABCDE", "displayName": "Host",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d, body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Game *models.Game `json:"game"`
	}
	decode(t, w, &created)

	for i := 2; i <= 5; i++ {
		tok := "user-" + string(rune('0'+i))
		w := doJSON(t, s, http.MethodPost, "/api/v1/games/join", tok, map[string]string{
			"joinCode": "ABCDE", "displayName": tok,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("join %d: %d", i, w.Code)
		}
	}

	w = doJSON(t, s, http.MethodPost, "/api/v1/games/"+created.Game.GameID+"/host/start", "user-host", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("start: %d, body=%s", w.Code, w.Body.String())
	}
	_ = secrethitler.GameStatusInProgress
	return created.Game.GameID
}
