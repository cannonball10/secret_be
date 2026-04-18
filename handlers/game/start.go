package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// CreateGame creates a new lobby hosted by the given user. The caller
// should generate a short join code (e.g. 5 uppercase letters) that
// players type into the mobile app. Collisions are left to the caller
// to retry; the database layer will reject duplicates via the join-code
// GSI in production.
func (h *GameHandler) CreateGame(ctx context.Context, hostUserID, joinCode, hostDisplayName string) (*models.Game, *models.Player, error) {
	joinCode = strings.ToUpper(strings.TrimSpace(joinCode))
	if joinCode == "" {
		return nil, nil, fmt.Errorf("%w: joinCode required", ErrInvalidTransition)
	}
	game := models.NewGame(nil, joinCode, hostUserID)

	// Host is implicitly the first player in the lobby at seat 0.
	host := models.NewPlayer(nil, game.GameID, hostUserID, hostDisplayName, true)
	host.Seat = 0

	if err := h.saveGame(ctx, game); err != nil {
		return nil, nil, err
	}
	if err := h.db.Upsert(ctx, nil, host); err != nil {
		return nil, nil, err
	}

	createdEvent := models.NewGameEvent(game.GameID, secrethitler.EventGameCreated, hostUserID)
	h.broadcast(ctx, createdEvent, GameCreatedPayload{JoinCode: joinCode, HostUserID: hostUserID})

	joinedEvent := models.NewGameEvent(game.GameID, secrethitler.EventPlayerJoined, host.PlayerID)
	h.broadcast(ctx, joinedEvent, PlayerJoinedPayload{
		PlayerID:    host.PlayerID,
		DisplayName: host.DisplayName,
		Seat:        host.Seat,
	})

	return game, host, nil
}

// JoinGame adds a player to a lobby keyed by join code. Rejects joins
// once the game is past the lobby phase or already at max players.
func (h *GameHandler) JoinGame(ctx context.Context, joinCode, userID, displayName string) (*models.Game, *models.Player, error) {
	game, err := h.lookupByJoinCode(ctx, joinCode)
	if err != nil {
		return nil, nil, err
	}
	if game.Status != secrethitler.GameStatusLobby {
		return nil, nil, fmt.Errorf("%w: game is not accepting joins", ErrInvalidTransition)
	}

	players, err := h.loadPlayers(ctx, game.GameID)
	if err != nil {
		return nil, nil, err
	}
	if len(players) >= secrethitler.MaxPlayers {
		return nil, nil, ErrTooManyPlayers
	}
	// Dedupe by userId so refreshing the mobile app doesn't seat twice.
	for _, p := range players {
		if p.UserID == userID {
			return game, p, nil
		}
	}

	player := models.NewPlayer(nil, game.GameID, userID, displayName, false)
	player.Seat = len(players)
	if err := h.db.Upsert(ctx, nil, player); err != nil {
		return nil, nil, err
	}

	ev := models.NewGameEvent(game.GameID, secrethitler.EventPlayerJoined, player.PlayerID)
	h.broadcast(ctx, ev, PlayerJoinedPayload{
		PlayerID:    player.PlayerID,
		DisplayName: player.DisplayName,
		Seat:        player.Seat,
	})
	return game, player, nil
}

// lookupByJoinCode looks up a Game via the JOIN# GSI.
func (h *GameHandler) lookupByJoinCode(ctx context.Context, joinCode string) (*models.Game, error) {
	indexName := database.IndexName_GSI1
	prefix := "GAME#"
	out, err := h.db.Query(ctx, nil, database.QueryInput{
		PartitionKey: models.GameGSI1Keys.PK(strings.ToUpper(joinCode)),
		SortKey:      &database.SortKeyCondition{BeginsWith: &prefix},
		IndexName:    &indexName,
	}, database.QueryOptions{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(out.Models) == 0 {
		return nil, ErrGameNotFound
	}
	return out.Models[0].(*models.Game), nil
}

// StartGame transitions a lobby into an in-progress game. Only the host
// may call this. Side effects:
//   - Deck is built and shuffled.
//   - Roles (liberal / fascist / hitler) are dealt based on player count
//     and whispered individually to each player.
//   - A random seat is chosen as the initial president.
//   - Phase advances to Nomination.
func (h *GameHandler) StartGame(ctx context.Context, gameID, hostUserID string) (*models.Game, error) {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game.HostUserID != hostUserID {
		return nil, ErrNotHost
	}
	if game.Status != secrethitler.GameStatusLobby {
		return nil, fmt.Errorf("%w: game already started", ErrInvalidTransition)
	}

	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if len(players) < secrethitler.MinPlayers {
		return nil, ErrNotEnoughPlayers
	}
	if len(players) > secrethitler.MaxPlayers {
		return nil, ErrTooManyPlayers
	}

	// Deal roles.
	if err := h.dealRoles(ctx, game, players); err != nil {
		return nil, err
	}

	// Build the deck.
	h.initDeck(game)

	// Randomly pick the starting president from among the seats.
	initialSeat := h.rng.IntN(len(players))
	game.Status = secrethitler.GameStatusInProgress
	game.PresidentSeat = initialSeat
	game.PlayerCount = len(players)
	now := h.clock.Now()
	game.StartedAt = &now
	game.Round = 1

	// Broadcast start and move to nomination. setPhase handles the
	// phase-changed emission and deadline.
	started := models.NewGameEvent(gameID, secrethitler.EventGameStarted, hostUserID)
	h.broadcast(ctx, started, GameStartedPayload{
		PlayerCount:          len(players),
		InitialPresidentSeat: initialSeat,
	})

	h.setPhase(ctx, game, secrethitler.PhaseNomination, ReasonAction)
	if err := h.saveGame(ctx, game); err != nil {
		return nil, err
	}
	// Bulk-save player updates (role assignment mutated them).
	items := make([]models.Model, 0, len(players))
	for _, p := range players {
		items = append(items, p)
	}
	if err := h.db.BulkUpsert(ctx, nil, items, nil); err != nil {
		return nil, err
	}
	return game, nil
}

// dealRoles assigns roles to the players using the configured RNG and
// whispers each player their private role (plus teammates where the
// rules require).
func (h *GameHandler) dealRoles(ctx context.Context, game *models.Game, players []*models.Player) error {
	liberals, fascists, hitlers, ok := secrethitler.RoleDistribution(len(players))
	if !ok {
		return ErrNotEnoughPlayers
	}

	roles := make([]secrethitler.Role, 0, liberals+fascists+hitlers)
	for i := 0; i < liberals; i++ {
		roles = append(roles, secrethitler.RoleLiberal)
	}
	for i := 0; i < fascists; i++ {
		roles = append(roles, secrethitler.RoleFascist)
	}
	for i := 0; i < hitlers; i++ {
		roles = append(roles, secrethitler.RoleHitler)
	}
	h.rng.Shuffle(len(roles), func(i, j int) { roles[i], roles[j] = roles[j], roles[i] })

	for i, p := range players {
		p.AssignRole(roles[i])
	}

	// In 5-6 player games Hitler sees the other fascist; in 7+ Hitler
	// sees nothing. Fascists always see each other and Hitler.
	fascistView := make([]TeammateInfo, 0)
	var hitler *models.Player
	for _, p := range players {
		switch p.Role {
		case secrethitler.RoleFascist, secrethitler.RoleHitler:
			fascistView = append(fascistView, TeammateInfo{
				PlayerID: p.PlayerID, DisplayName: p.DisplayName, Role: p.Role,
			})
		}
		if p.IsHitler() {
			hitler = p
		}
	}

	// Broadcast the "roles assigned" event so clients know to prompt
	// each player for their private reveal.
	h.broadcast(ctx, models.NewGameEvent(game.GameID, secrethitler.EventRolesAssigned, ""), nil)

	// Whisper to each player.
	for _, p := range players {
		payload := RoleAssignedPayload{Role: p.Role, Party: p.Party}
		switch p.Role {
		case secrethitler.RoleFascist:
			payload.Teammates = fascistView
		case secrethitler.RoleHitler:
			if len(players) <= 6 && hitler != nil {
				payload.Teammates = fascistView
			}
		}
		ev := models.NewGameEvent(game.GameID, secrethitler.EventRolesAssigned, p.PlayerID)
		h.whisper(ctx, ev, p.PlayerID, payload)
	}
	return nil
}
