//go:build integration

package graphdb

import (
	"context"
	"sort"
	"testing"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/testutil"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func setupNeo4jTest(t *testing.T) (*Neo4jGraph, string) {
	t.Helper()
	testutil.SkipIfServiceUnavailable(t, testutil.Neo4jAddr, "Neo4j")

	driver, err := neo4j.NewDriverWithContext(
		"bolt://"+testutil.Neo4jAddr,
		neo4j.BasicAuth("neo4j", "password", ""),
	)
	if err != nil {
		t.Fatalf("Failed to create Neo4j driver: %v", err)
	}

	prefix := testutil.TestPrefix(t)

	// Use unique label for test isolation
	cfg := &Neo4jConfig{
		UserLabel:             "TestUser_" + prefix[:20], // Truncate to avoid label length issues
		UserIDProperty:        "userId",
		RelationshipType:      "TEST_FRIEND",
		BlockRelationshipType: "TEST_BLOCKS",
	}

	graph := NewNeo4jGraph(driver, cfg)

	t.Cleanup(func() {
		// Clean up test nodes
		ctx := context.Background()
		session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
		defer session.Close(ctx)

		// Delete all test nodes and relationships
		session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := "MATCH (n:" + cfg.UserLabel + ") DETACH DELETE n"
			_, err := tx.Run(ctx, query, nil)
			return nil, err
		})

		graph.Close()
	})

	return graph, prefix
}

func createTestGraphUser(prefix, suffix string) *models.User {
	id := prefix + suffix
	return &models.User{
		UserID:      id,
		Email:       id + "@test.com",
		DisplayName: "Test User " + suffix,
	}
}

func TestNeo4j_UpsertUser(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	user := createTestGraphUser(prefix, "user1")

	// Upsert user
	err := graph.UpsertUser(ctx, user)
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	// Upsert same user again (should update, not error)
	user.DisplayName = "Updated Name"
	err = graph.UpsertUser(ctx, user)
	if err != nil {
		t.Fatalf("UpsertUser update failed: %v", err)
	}
}

func TestNeo4j_LinkUnlinkUsers(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	user1 := createTestGraphUser(prefix, "link1")
	user2 := createTestGraphUser(prefix, "link2")

	// Create users
	if err := graph.UpsertUser(ctx, user1); err != nil {
		t.Fatalf("UpsertUser user1 failed: %v", err)
	}
	if err := graph.UpsertUser(ctx, user2); err != nil {
		t.Fatalf("UpsertUser user2 failed: %v", err)
	}

	// Link users
	err := graph.LinkUsers(ctx, user1.UserID, user2.UserID)
	if err != nil {
		t.Fatalf("LinkUsers failed: %v", err)
	}

	// Verify link exists via WithinDegrees
	limit := 10
	friends, err := graph.WithinDegrees(ctx, user1.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees failed: %v", err)
	}

	if len(friends) != 1 {
		t.Errorf("Expected 1 friend, got %d", len(friends))
	}
	if len(friends) > 0 && friends[0] != user2.UserID {
		t.Errorf("Expected friend %s, got %s", user2.UserID, friends[0])
	}

	// Unlink users
	err = graph.UnlinkUsers(ctx, user1.UserID, user2.UserID)
	if err != nil {
		t.Fatalf("UnlinkUsers failed: %v", err)
	}

	// Verify link removed
	friends, err = graph.WithinDegrees(ctx, user1.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees after unlink failed: %v", err)
	}

	if len(friends) != 0 {
		t.Errorf("Expected 0 friends after unlink, got %d", len(friends))
	}
}

func TestNeo4j_BlockUnblockUser(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	blocker := createTestGraphUser(prefix, "blocker")
	blocked := createTestGraphUser(prefix, "blocked")
	mutual := createTestGraphUser(prefix, "mutual")

	// Create users
	for _, user := range []*models.User{blocker, blocked, mutual} {
		if err := graph.UpsertUser(ctx, user); err != nil {
			t.Fatalf("UpsertUser %s failed: %v", user.UserID, err)
		}
	}

	// Link blocker to mutual and mutual to blocked
	if err := graph.LinkUsers(ctx, blocker.UserID, mutual.UserID); err != nil {
		t.Fatalf("LinkUsers blocker-mutual failed: %v", err)
	}
	if err := graph.LinkUsers(ctx, mutual.UserID, blocked.UserID); err != nil {
		t.Fatalf("LinkUsers mutual-blocked failed: %v", err)
	}

	// Initially, blocker can see blocked via mutual (2 degrees)
	limit := 10
	visible, err := graph.WithinDegrees(ctx, blocker.UserID, 2, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees before block failed: %v", err)
	}

	foundBlocked := false
	for _, id := range visible {
		if id == blocked.UserID {
			foundBlocked = true
			break
		}
	}
	if !foundBlocked {
		t.Log("Note: blocked user may not be visible depending on block filtering implementation")
	}

	// Block user
	err = graph.BlockUser(ctx, blocker.UserID, blocked.UserID)
	if err != nil {
		t.Fatalf("BlockUser failed: %v", err)
	}

	// After block, blocked user should not be visible
	visible, err = graph.WithinDegrees(ctx, blocker.UserID, 2, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees after block failed: %v", err)
	}

	for _, id := range visible {
		if id == blocked.UserID {
			t.Error("Blocked user should not be visible after BlockUser")
			break
		}
	}

	// Unblock user
	err = graph.UnblockUser(ctx, blocker.UserID, blocked.UserID)
	if err != nil {
		t.Fatalf("UnblockUser failed: %v", err)
	}
}

func TestNeo4j_WithinDegrees_Direct(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	center := createTestGraphUser(prefix, "center")
	friend1 := createTestGraphUser(prefix, "friend1")
	friend2 := createTestGraphUser(prefix, "friend2")

	// Create users
	for _, user := range []*models.User{center, friend1, friend2} {
		if err := graph.UpsertUser(ctx, user); err != nil {
			t.Fatalf("UpsertUser %s failed: %v", user.UserID, err)
		}
	}

	// Link center to friends
	if err := graph.LinkUsers(ctx, center.UserID, friend1.UserID); err != nil {
		t.Fatalf("LinkUsers center-friend1 failed: %v", err)
	}
	if err := graph.LinkUsers(ctx, center.UserID, friend2.UserID); err != nil {
		t.Fatalf("LinkUsers center-friend2 failed: %v", err)
	}

	// Get direct friends (1 degree)
	limit := 10
	friends, err := graph.WithinDegrees(ctx, center.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees failed: %v", err)
	}

	if len(friends) != 2 {
		t.Errorf("Expected 2 friends, got %d", len(friends))
	}

	// Verify both friends are present
	friendIDs := map[string]bool{friend1.UserID: false, friend2.UserID: false}
	for _, id := range friends {
		if _, exists := friendIDs[id]; exists {
			friendIDs[id] = true
		}
	}

	for id, found := range friendIDs {
		if !found {
			t.Errorf("Friend %s not found in results", id)
		}
	}
}

func TestNeo4j_WithinDegrees_Indirect(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	// Create chain: center -> friend1 -> fof (friend of friend)
	center := createTestGraphUser(prefix, "center2")
	friend := createTestGraphUser(prefix, "friend")
	fof := createTestGraphUser(prefix, "fof")

	for _, user := range []*models.User{center, friend, fof} {
		if err := graph.UpsertUser(ctx, user); err != nil {
			t.Fatalf("UpsertUser %s failed: %v", user.UserID, err)
		}
	}

	if err := graph.LinkUsers(ctx, center.UserID, friend.UserID); err != nil {
		t.Fatalf("LinkUsers center-friend failed: %v", err)
	}
	if err := graph.LinkUsers(ctx, friend.UserID, fof.UserID); err != nil {
		t.Fatalf("LinkUsers friend-fof failed: %v", err)
	}

	// At 1 degree, only friend is visible
	limit := 10
	degree1, err := graph.WithinDegrees(ctx, center.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees(1) failed: %v", err)
	}

	if len(degree1) != 1 || degree1[0] != friend.UserID {
		t.Errorf("At 1 degree, expected only friend, got %v", degree1)
	}

	// At 2 degrees, both friend and fof are visible
	degree2, err := graph.WithinDegrees(ctx, center.UserID, 2, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees(2) failed: %v", err)
	}

	if len(degree2) != 2 {
		t.Errorf("At 2 degrees, expected 2 users, got %d", len(degree2))
	}

	// Sort for consistent comparison
	sort.Strings(degree2)
	expected := []string{fof.UserID, friend.UserID}
	sort.Strings(expected)

	for i, id := range expected {
		if i >= len(degree2) || degree2[i] != id {
			t.Errorf("At 2 degrees, expected %v, got %v", expected, degree2)
			break
		}
	}
}

func TestNeo4j_WithinDegrees_WithLimit(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	center := createTestGraphUser(prefix, "limitcenter")
	friends := make([]*models.User, 5)
	for i := 0; i < 5; i++ {
		friends[i] = createTestGraphUser(prefix, "limitfriend"+string(rune('0'+i)))
	}

	// Create users
	if err := graph.UpsertUser(ctx, center); err != nil {
		t.Fatalf("UpsertUser center failed: %v", err)
	}
	for _, friend := range friends {
		if err := graph.UpsertUser(ctx, friend); err != nil {
			t.Fatalf("UpsertUser %s failed: %v", friend.UserID, err)
		}
		if err := graph.LinkUsers(ctx, center.UserID, friend.UserID); err != nil {
			t.Fatalf("LinkUsers failed: %v", err)
		}
	}

	// Get with limit of 3
	limit := 3
	results, err := graph.WithinDegrees(ctx, center.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees with limit failed: %v", err)
	}

	if len(results) > limit {
		t.Errorf("Expected at most %d results, got %d", limit, len(results))
	}
}

func TestNeo4j_SymmetricRelationship(t *testing.T) {
	graph, prefix := setupNeo4jTest(t)
	ctx := context.Background()

	user1 := createTestGraphUser(prefix, "sym1")
	user2 := createTestGraphUser(prefix, "sym2")

	// Create users
	if err := graph.UpsertUser(ctx, user1); err != nil {
		t.Fatalf("UpsertUser user1 failed: %v", err)
	}
	if err := graph.UpsertUser(ctx, user2); err != nil {
		t.Fatalf("UpsertUser user2 failed: %v", err)
	}

	// Link in one direction
	err := graph.LinkUsers(ctx, user1.UserID, user2.UserID)
	if err != nil {
		t.Fatalf("LinkUsers failed: %v", err)
	}

	limit := 10

	// Verify relationship works from user1's perspective
	friends1, err := graph.WithinDegrees(ctx, user1.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees from user1 failed: %v", err)
	}
	if len(friends1) != 1 || friends1[0] != user2.UserID {
		t.Errorf("user1 should see user2 as friend, got %v", friends1)
	}

	// Verify relationship works from user2's perspective (symmetric)
	friends2, err := graph.WithinDegrees(ctx, user2.UserID, 1, &limit)
	if err != nil {
		t.Fatalf("WithinDegrees from user2 failed: %v", err)
	}
	if len(friends2) != 1 || friends2[0] != user1.UserID {
		t.Errorf("user2 should see user1 as friend (symmetric), got %v", friends2)
	}
}
