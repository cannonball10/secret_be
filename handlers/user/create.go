package user

import (
	"context"

	"github.com/cannonball10/foundation/models"
)

func (h *UserHandler) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	err := h.deps.GetConnectors().Database.Upsert(ctx, nil, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
