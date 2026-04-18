package graphdb

import (
	"context"

	"github.com/cannonball10/foundation/models"
)

// GraphDBConnector is an abstract graph database interface that can be implemented by
// concrete providers like Neo4j, etc.
//
// Relationships:
//   - LinkUsers/UnlinkUsers represent an "undirected" user relationship (e.g. FRIEND). Implementations
//     may store this as a directed edge using canonical ordering to prevent duplicates.
//   - BlockUser/UnblockUser are directed ("A blocks B").
//
// Traversal:
// - WithinDegrees returns user IDs within <= maxDegrees hops from startUserID, excluding startUserID.
type GraphDBConnector interface {
	UpsertUser(ctx context.Context, user *models.User) error

	LinkUsers(ctx context.Context, userID1, userID2 string) error
	UnlinkUsers(ctx context.Context, userID1, userID2 string) error

	BlockUser(ctx context.Context, blockerUserID, blockedUserID string) error
	UnblockUser(ctx context.Context, blockerUserID, blockedUserID string) error

	WithinDegrees(ctx context.Context, startUserID string, maxDegrees int, limit *int) ([]string, error)
}

func DefaultGraphDBConnector(ctx context.Context) (GraphDBConnector, error) {
	return DefaultNeo4jGraphConnector(ctx)
}
