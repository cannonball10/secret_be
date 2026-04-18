package graphdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/graphdb"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var _ GraphDBConnector = (*Neo4jGraph)(nil)

var errNeo4jDriverUninitialized = errors.New("neo4j driver is not initialized")

// Neo4jConfig configures the Neo4j-backed GraphDB connector. Nil/empty values use defaults.
type Neo4jConfig struct {
	// Database is the Neo4j database name. Empty uses the server default.
	Database string

	// UserLabel is the node label for users (default "User").
	UserLabel string
	// UserIDProperty is the unique user ID property name (default "userId").
	UserIDProperty string

	// RelationshipType is the (single) relationship type used for the undirected user relationship
	// (default "FRIEND"). Implementations may store it as directed via canonical ordering.
	RelationshipType string

	// BlockRelationshipType is the directed relationship type representing a block
	// (default "BLOCKS").
	BlockRelationshipType string
}

// Neo4jGraph implements GraphDBConnector using the Neo4j Go driver (v5).
type Neo4jGraph struct {
	driver        neo4j.DriverWithContext
	driverOnce    sync.Once
	driverFactory func() neo4j.DriverWithContext

	database  string
	userLabel string
	userIDKey string

	relType      string
	blockRelType string

	cfgErr error
}

// NewNeo4jGraph returns a connector with an eagerly provided driver.
func NewNeo4jGraph(driver neo4j.DriverWithContext, cfg *Neo4jConfig) *Neo4jGraph {
	return newNeo4jGraph(driver, nil, cfg)
}

// NewNeo4jGraphLazy returns a connector that initializes the driver on first use.
func NewNeo4jGraphLazy(factory func() neo4j.DriverWithContext, cfg *Neo4jConfig) *Neo4jGraph {
	return newNeo4jGraph(nil, factory, cfg)
}

func DefaultNeo4jGraphConnector(ctx context.Context) (*Neo4jGraph, error) {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		return nil, errors.New("NEO4J_URI is not set")
	}
	user := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")
	auth := neo4j.BasicAuth(user, password, "")
	if user == "" || password == "" {
		auth = neo4j.NoAuth()
	}
	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return nil, err
	}
	return NewNeo4jGraph(driver, nil), nil
}

func newNeo4jGraph(driver neo4j.DriverWithContext, factory func() neo4j.DriverWithContext, cfg *Neo4jConfig) *Neo4jGraph {
	c := applyNeo4jDefaults(cfg)
	g := &Neo4jGraph{
		driver:        driver,
		driverFactory: factory,

		database:     c.Database,
		userLabel:    c.UserLabel,
		userIDKey:    c.UserIDProperty,
		relType:      c.RelationshipType,
		blockRelType: c.BlockRelationshipType,
	}

	// Validate identifiers once; surface as runtime errors from methods.
	g.cfgErr = validateNeo4jIdentifiers(g.userLabel, g.userIDKey, g.relType, g.blockRelType)
	return g
}

func applyNeo4jDefaults(cfg *Neo4jConfig) Neo4jConfig {
	if cfg == nil {
		return Neo4jConfig{
			Database:              "",
			UserLabel:             graphdb.DefaultUserLabel,
			UserIDProperty:        graphdb.DefaultUserIDProperty,
			RelationshipType:      graphdb.DefaultRelationshipType,
			BlockRelationshipType: graphdb.DefaultBlockRelationshipType,
		}
	}

	out := *cfg
	if out.UserLabel == "" {
		out.UserLabel = graphdb.DefaultUserLabel
	}
	if out.UserIDProperty == "" {
		out.UserIDProperty = graphdb.DefaultUserIDProperty
	}
	if out.RelationshipType == "" {
		out.RelationshipType = graphdb.DefaultRelationshipType
	}
	if out.BlockRelationshipType == "" {
		out.BlockRelationshipType = graphdb.DefaultBlockRelationshipType
	}
	return out
}

func validateNeo4jIdentifiers(userLabel, userIDKey, relType, blockRelType string) error {
	for name, value := range map[string]string{
		"UserLabel":             userLabel,
		"UserIDProperty":        userIDKey,
		"RelationshipType":      relType,
		"BlockRelationshipType": blockRelType,
	} {
		if !isSafeCypherIdent(value) {
			return fmt.Errorf("neo4j config %s=%q is not a safe Cypher identifier", name, value)
		}
	}
	return nil
}

// isSafeCypherIdent is a conservative check to prevent Cypher injection when we embed
// labels, property keys, and relationship types into query strings. It allows only
// ASCII letters, digits, and underscores, and requires the first char to be a letter
// or underscore.
func isSafeCypherIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		isLetter := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
		isDigit := b >= '0' && b <= '9'
		isUnderscore := b == '_'

		if i == 0 {
			if !(isLetter || isUnderscore) {
				return false
			}
			continue
		}
		if !(isLetter || isDigit || isUnderscore) {
			return false
		}
	}
	return true
}

func (g *Neo4jGraph) getDriver() (neo4j.DriverWithContext, error) {
	g.driverOnce.Do(func() {
		if g.driver == nil && g.driverFactory != nil {
			g.driver = g.driverFactory()
		}
	})
	if g.driver == nil {
		return nil, errNeo4jDriverUninitialized
	}
	return g.driver, nil
}

// Ping verifies Neo4j connectivity. Implements connectors.ConnectorHealthChecker.
func (g *Neo4jGraph) Ping(ctx context.Context) error {
	driver, err := g.getDriver()
	if err != nil {
		return err
	}
	return driver.VerifyConnectivity(ctx)
}

// Close shuts down the Neo4j driver. Implements connectors.ConnectorCloser.
func (g *Neo4jGraph) Close() error {
	// Don't force-initialize on Close.
	if g.driver == nil {
		return nil
	}
	return g.driver.Close(context.Background())
}

func orderedPair(a, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}

func (g *Neo4jGraph) newSession(ctx context.Context, mode neo4j.AccessMode) (neo4j.SessionWithContext, error) {
	if g.cfgErr != nil {
		return nil, g.cfgErr
	}
	driver, err := g.getDriver()
	if err != nil {
		return nil, err
	}

	cfg := neo4j.SessionConfig{
		AccessMode: mode,
	}
	if g.database != "" {
		cfg.DatabaseName = g.database
	}
	return driver.NewSession(ctx, cfg), nil
}

func (g *Neo4jGraph) UpsertUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.UserID == "" {
		return errors.New("user.userId cannot be empty")
	}

	now := time.Now().UTC()
	createdAt := user.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := user.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	}

	session, err := g.newSession(ctx, neo4j.AccessModeWrite)
	if err != nil {
		return err
	}
	defer session.Close(ctx)

	query := fmt.Sprintf(
		`MERGE (u:%s {%s: $userId})
SET u.email = $email,
    u.displayName = $displayName,
    u.updatedAt = $updatedAt,
    u.createdAt = coalesce(u.createdAt, $createdAt)`,
		g.userLabel,
		g.userIDKey,
	)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, runErr := tx.Run(ctx, query, map[string]any{
			"userId":      user.UserID,
			"email":       user.Email,
			"displayName": user.DisplayName,
			"createdAt":   createdAt,
			"updatedAt":   updatedAt,
		})
		return nil, runErr
	})
	if err != nil {
		slog.ErrorContext(ctx, "neo4j UpsertUser failed", "operation", "UpsertUser", "user_id", user.UserID, "error", err)
	}
	return err
}

func (g *Neo4jGraph) LinkUsers(ctx context.Context, userID1, userID2 string) error {
	if userID1 == "" || userID2 == "" {
		return errors.New("user IDs cannot be empty")
	}
	if userID1 == userID2 {
		return errors.New("cannot link a user to itself")
	}

	a, b := orderedPair(userID1, userID2)

	session, err := g.newSession(ctx, neo4j.AccessModeWrite)
	if err != nil {
		return err
	}
	defer session.Close(ctx)

	query := fmt.Sprintf(
		`MERGE (a:%s {%s: $a})
MERGE (b:%s {%s: $b})
MERGE (a)-[:%s]->(b)`,
		g.userLabel, g.userIDKey,
		g.userLabel, g.userIDKey,
		g.relType,
	)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, runErr := tx.Run(ctx, query, map[string]any{
			"a": a,
			"b": b,
		})
		return nil, runErr
	})
	if err != nil {
		slog.ErrorContext(ctx, "neo4j LinkUsers failed", "operation", "LinkUsers", "user_id_1", userID1, "user_id_2", userID2, "error", err)
	}
	return err
}

func (g *Neo4jGraph) UnlinkUsers(ctx context.Context, userID1, userID2 string) error {
	if userID1 == "" || userID2 == "" {
		return errors.New("user IDs cannot be empty")
	}
	if userID1 == userID2 {
		return nil
	}

	a, b := orderedPair(userID1, userID2)

	session, err := g.newSession(ctx, neo4j.AccessModeWrite)
	if err != nil {
		return err
	}
	defer session.Close(ctx)

	query := fmt.Sprintf(
		`MATCH (a:%s {%s: $a})-[r:%s]->(b:%s {%s: $b})
DELETE r`,
		g.userLabel, g.userIDKey,
		g.relType,
		g.userLabel, g.userIDKey,
	)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, runErr := tx.Run(ctx, query, map[string]any{
			"a": a,
			"b": b,
		})
		return nil, runErr
	})
	if err != nil {
		slog.ErrorContext(ctx, "neo4j UnlinkUsers failed", "operation", "UnlinkUsers", "user_id_1", userID1, "user_id_2", userID2, "error", err)
	}
	return err
}

func (g *Neo4jGraph) BlockUser(ctx context.Context, blockerUserID, blockedUserID string) error {
	if blockerUserID == "" || blockedUserID == "" {
		return errors.New("user IDs cannot be empty")
	}
	if blockerUserID == blockedUserID {
		return errors.New("cannot block a user from itself")
	}

	session, err := g.newSession(ctx, neo4j.AccessModeWrite)
	if err != nil {
		return err
	}
	defer session.Close(ctx)

	query := fmt.Sprintf(
		`MERGE (a:%s {%s: $blocker})
MERGE (b:%s {%s: $blocked})
MERGE (a)-[:%s]->(b)`,
		g.userLabel, g.userIDKey,
		g.userLabel, g.userIDKey,
		g.blockRelType,
	)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, runErr := tx.Run(ctx, query, map[string]any{
			"blocker": blockerUserID,
			"blocked": blockedUserID,
		})
		return nil, runErr
	})
	if err != nil {
		slog.ErrorContext(ctx, "neo4j BlockUser failed", "operation", "BlockUser", "blocker_user_id", blockerUserID, "blocked_user_id", blockedUserID, "error", err)
	}
	return err
}

func (g *Neo4jGraph) UnblockUser(ctx context.Context, blockerUserID, blockedUserID string) error {
	if blockerUserID == "" || blockedUserID == "" {
		return errors.New("user IDs cannot be empty")
	}
	if blockerUserID == blockedUserID {
		return nil
	}

	session, err := g.newSession(ctx, neo4j.AccessModeWrite)
	if err != nil {
		return err
	}
	defer session.Close(ctx)

	query := fmt.Sprintf(
		`MATCH (a:%s {%s: $blocker})-[r:%s]->(b:%s {%s: $blocked})
DELETE r`,
		g.userLabel, g.userIDKey,
		g.blockRelType,
		g.userLabel, g.userIDKey,
	)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, runErr := tx.Run(ctx, query, map[string]any{
			"blocker": blockerUserID,
			"blocked": blockedUserID,
		})
		return nil, runErr
	})
	if err != nil {
		slog.ErrorContext(ctx, "neo4j UnblockUser failed", "operation", "UnblockUser", "blocker_user_id", blockerUserID, "blocked_user_id", blockedUserID, "error", err)
	}
	return err
}

func (g *Neo4jGraph) WithinDegrees(ctx context.Context, startUserID string, maxDegrees int, limit *int) ([]string, error) {
	if startUserID == "" {
		return nil, errors.New("startUserID cannot be empty")
	}
	if maxDegrees <= 0 {
		return []string{}, nil
	}

	session, err := g.newSession(ctx, neo4j.AccessModeRead)
	if err != nil {
		return nil, err
	}
	defer session.Close(ctx)

	// NOTE: Neo4j Cypher does not reliably allow a parameter for variable-length bounds,
	// so we embed maxDegrees as an integer literal.
	baseQuery := fmt.Sprintf(
		`MATCH (start:%s {%s: $start})
MATCH p=(start)-[:%s*1..%d]-(other:%s)
WHERE other.%s <> $start
  AND NOT EXISTS { MATCH (start)-[:%s]->(other) }
  AND NOT EXISTS { MATCH (other)-[:%s]->(start) }
  AND NONE(r IN relationships(p) WHERE
        EXISTS {
              WITH startNode(r) AS a, endNode(r) AS b
              MATCH (a)-[:%s]-(b)
        }
  )
RETURN DISTINCT other.%s AS userId`,
		g.userLabel, g.userIDKey,
		g.relType, maxDegrees,
		g.userLabel,
		g.userIDKey,
		g.blockRelType,
		g.blockRelType,
		g.blockRelType,
		g.userIDKey,
	)

	params := map[string]any{
		"start": startUserID,
	}

	query := baseQuery
	if limit != nil && *limit > 0 {
		query = baseQuery + "\nLIMIT $limit"
		params["limit"] = *limit
	}

	outAny, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, runErr := tx.Run(ctx, query, params)
		if runErr != nil {
			return nil, runErr
		}
		ids := make([]string, 0)
		for res.Next(ctx) {
			rec := res.Record()
			v, ok := rec.Get("userId")
			if ok {
				if s, ok := v.(string); ok && s != "" && s != startUserID {
					ids = append(ids, s)
				}
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return ids, nil
	})
	if err != nil {
		slog.ErrorContext(ctx, "neo4j WithinDegrees failed", "operation", "WithinDegrees", "user_id", startUserID, "error", err)
		return nil, err
	}

	ids, _ := outAny.([]string)
	// De-dupe defensively while preserving order (Cypher DISTINCT should already do this).
	seen := map[string]struct{}{}
	final := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || id == startUserID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		final = append(final, id)
	}
	return final, nil
}
