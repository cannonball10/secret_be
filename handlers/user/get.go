package user

import (
	"context"

	"github.com/cannonball10/foundation/models"
)

func (h *UserHandler) GetUser(ctx context.Context, userID string) (*models.User, error) {
	result, err := h.deps.GetConnectors().Database.Get(ctx, nil, models.UserKeys.Key(userID))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.User), nil
}
