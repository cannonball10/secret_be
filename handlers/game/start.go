package game

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// CreateGame creates a new lobby hosted by the given user.
//
// joinCode is optional: pass "" to have the engine generate a random
// 6-char alphanumeric code. Non-empty codes are uppercased and used
// verbatim — collisions are the caller's problem (the join-code GSI
// enforces uniqueness in production).
//
// hostDisplayName is optional: pass "" to create a "board-only" lobby
// where the host device is a spectator (no Player record, not seated,
// not dealt a role). Players join separately via JoinGame. When non-
// empty the host is seated at seat 0 — the classic self-as-first-player
// flow — and the returned *models.Player is non-nil.
func (h *GameHandler) CreateGame(ctx context.Context, hostUserID, joinCode, hostDisplayName string) (*models.Game, *models.Player, error) {
	joinCode = strings.ToUpper(strings.TrimSpace(joinCode))
	if joinCode == "" {
		joinCode = h.generateJoinCode()
	}
	game := models.NewGame(nil, joinCode, hostUserID)

	if err := h.saveGame(ctx, game); err != nil {
		return nil, nil, err
	}

	createdEvent := models.NewGameEvent(game.GameID, secrethitler.EventGameCreated, hostUserID)
	h.broadcast(ctx, createdEvent, GameCreatedPayload{JoinCode: joinCode, HostUserID: hostUserID})

	// Board-only host: return early with a nil Player.
	if strings.TrimSpace(hostDisplayName) == "" {
		return game, nil, nil
	}

	host := models.NewPlayer(nil, game.GameID, hostUserID, hostDisplayName, true)
	host.Seat = 0
	if err := h.db.Upsert(ctx, nil, host); err != nil {
		return nil, nil, err
	}

	joinedEvent := models.NewGameEvent(game.GameID, secrethitler.EventPlayerJoined, host.PlayerID)
	h.broadcast(ctx, joinedEvent, PlayerJoinedPayload{
		PlayerID:    host.PlayerID,
		DisplayName: host.DisplayName,
		Seat:        host.Seat,
	})

	return game, host, nil
}

// joinCodeAlphabet intentionally excludes ambiguous glyphs (0/O, 1/I/L)
// so hosts can read the code aloud without players mistyping it.
const joinCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generateJoinCode returns a 6-character alphanumeric code using the
// engine's RNG so tests can pin it via a seeded RNG.
func (h *GameHandler) generateJoinCode() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = joinCodeAlphabet[h.rng.IntN(len(joinCodeAlphabet))]
	}
	return string(b)
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
	// Belt-and-suspenders: defend against legacy Game records where
	// Rules.MaxPlayers round-trips as a bogus low value (0 from a
	// missing field, or a partial-write leftover). ensureRules already
	// backfills these on load, but if something slips through we clamp
	// to the package maximum so a single bad record can't strand a
	// full-sized table.
	maxP := game.Rules.MaxPlayers
	if maxP <= 0 || maxP > secrethitler.PackageMaxPlayers {
		slog.Warn("joinGame: Rules.MaxPlayers out of range, using package max",
			"gameId", game.GameID, "stored", maxP)
		maxP = secrethitler.PackageMaxPlayers
	}
	if len(players) >= maxP {
		slog.Info("joinGame: table full",
			"gameId", game.GameID, "players", len(players), "max", maxP,
			"storedMin", game.Rules.MinPlayers, "storedMax", game.Rules.MaxPlayers)
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
	g := out.Models[0].(*models.Game)
	ensureRules(g)
	return g, nil
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
	if len(players) < game.Rules.MinPlayers {
		return nil, ErrNotEnoughPlayers
	}
	if len(players) > game.Rules.MaxPlayers {
		return nil, ErrTooManyPlayers
	}

	// Deal roles.
	if err := h.dealRoles(ctx, game, players); err != nil {
		return nil, err
	}

	// Assign a country to each seated delegate. The CommitteeRoster
	// is seat-indexed so repeated games on the same table pick up the
	// same delegation in the same chair — useful for habitual groups
	// that think of "Priya plays France" as part of the identity.
	for _, p := range players {
		country := secrethitler.CountryForSeat(p.Seat)
		p.CountryCode = country.Code
		p.CountryName = country.Name
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
		Round:                game.Round,
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
// rules require). Honours Rules.EnableSingularity by picking the
// distribution table — either vanilla or Singularity-included.
func (h *GameHandler) dealRoles(ctx context.Context, game *models.Game, players []*models.Player) error {
	liberals, fascists, hitlers, singularities, ok := rolesSeating(game, len(players))
	if !ok {
		return ErrNotEnoughPlayers
	}

	roles := make([]secrethitler.Role, 0, liberals+fascists+hitlers+singularities)
	for i := 0; i < liberals; i++ {
		roles = append(roles, secrethitler.RoleHuman)
	}
	for i := 0; i < fascists; i++ {
		roles = append(roles, secrethitler.RoleAI)
	}
	for i := 0; i < hitlers; i++ {
		roles = append(roles, secrethitler.RoleRogue)
	}
	for i := 0; i < singularities; i++ {
		roles = append(roles, secrethitler.RoleSingularity)
	}
	h.rng.Shuffle(len(roles), func(i, j int) { roles[i], roles[j] = roles[j], roles[i] })

	for i, p := range players {
		p.AssignRole(roles[i])
	}

	// Broadcast the "roles assigned" event so clients know to prompt
	// each player for their private reveal.
	h.broadcast(ctx, models.NewGameEvent(game.GameID, secrethitler.EventRolesAssigned, ""), nil)

	// Whisper each player their private view. The reveal matrix:
	//
	//   AI cabal    : always see every other fascist and Hitler.
	//   Hitler      : in 5-6 player games, sees the single fascist.
	//                 In 7+ player games, sees nothing.
	//   Liberals    : see nothing.
	//   Singularity : sees nothing; plays alone by design.
	for _, p := range players {
		payload := RoleAssignedPayload{Role: p.Role, Party: p.Party}
		switch p.Role {
		case secrethitler.RoleAI:
			payload.Teammates = cabalViewExcluding(players, p.PlayerID)
		case secrethitler.RoleRogue:
			if len(players) <= 6 {
				payload.Teammates = cabalViewExcluding(players, p.PlayerID)
			}
		}
		ev := models.NewGameEvent(game.GameID, secrethitler.EventRolesAssigned, p.PlayerID)
		h.whisper(ctx, ev, p.PlayerID, payload)
	}
	return nil
}

// rolesSeating picks the role-distribution table to use based on the
// ruleset and returns the per-role seat counts. The Singularity column
// is always zero for the vanilla path.
func rolesSeating(game *models.Game, playerCount int) (liberals, fascists, hitlers, singularities int, ok bool) {
	if game.Rules.EnableSingularity {
		return secrethitler.RoleDistributionWithSingularity(playerCount)
	}
	l, f, h, ok := secrethitler.RoleDistribution(playerCount)
	return l, f, h, 0, ok
}

// cabalViewExcluding returns the list of fascist-aligned players (fascists
// and Hitler) excluding the player whose private view we're building.
func cabalViewExcluding(players []*models.Player, selfPlayerID string) []TeammateInfo {
	out := make([]TeammateInfo, 0)
	for _, p := range players {
		if p.PlayerID == selfPlayerID {
			continue
		}
		if p.Role == secrethitler.RoleAI || p.Role == secrethitler.RoleRogue {
			out = append(out, TeammateInfo{
				PlayerID:    p.PlayerID,
				DisplayName: p.DisplayName,
				Role:        p.Role,
			})
		}
	}
	return out
}
