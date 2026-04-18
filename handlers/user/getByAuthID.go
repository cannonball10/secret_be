package user

import (
	"context"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/authentication"
	"github.com/cannonball10/foundation/schemas/database"
)

// GetByAuthID looks up a user by authentication provider and external ID using GSI1.
// Returns (nil, nil) if no matching user is found.
func (h *UserHandler) GetByAuthID(ctx context.Context, provider authentication.AuthenticationProvider, authenticationID string) (*models.User, error) {
	indexName := database.IndexName_GSI1
	sk := models.UserGSI1Keys.SK(authenticationID)
	out, err := h.deps.GetConnectors().Database.Query(ctx, nil, database.QueryInput{
		PartitionKey: models.UserGSI1Keys.PK(string(provider)),
		SortKey:      &database.SortKeyCondition{EQ: &sk},
		IndexName:    &indexName,
	}, database.QueryOptions{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(out.Models) == 0 {
		return nil, nil
	}
	return out.Models[0].(*models.User), nil
}
