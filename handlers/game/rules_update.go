package game

import (
	"context"
	"fmt"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/replicant"
)

// UpdateRules replaces the game's RulesConfig before StartGame fires.
// Host-only, lobby-only. The caller sends a full RulesConfig (not a
// diff) — clients should fetch the current game, mutate one field on
// the returned Rules, and POST the whole thing back. This keeps the
// endpoint stateless and avoids the field-by-field merge ambiguity
// (e.g. "is CablePhaseDurationSec=0 'leave alone' or 'set to 0'?").
//
// Validation runs before persist. A rejected config returns the engine
// error without touching the stored game, so a bad edit can't strand
// a lobby.
func (h *GameHandler) UpdateRules(ctx context.Context, gameID, hostUserID string, rules replicant.RulesConfig) (*models.Game, error) {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game.HostUserID != hostUserID {
		return nil, ErrNotHost
	}
	if game.Status != replicant.GameStatusLobby {
		return nil, fmt.Errorf("%w: rules can only be edited in the lobby", ErrInvalidTransition)
	}
	if err := rules.Validate(); err != nil {
		// Surface validation errors to the caller so the UI can highlight
		// which constraint failed. Wrap with ErrInvalidTransition so the
		// HTTP layer maps it to 400 rather than 500.
		return nil, fmt.Errorf("%w: %s", ErrInvalidTransition, err.Error())
	}
	game.Rules = rules
	if err := h.saveGame(ctx, game); err != nil {
		return nil, err
	}
	return game, nil
}
