//go:build integration

package database

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/authentication"
	dbSchema "github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/testutil"
	"github.com/cannonball10/foundation/utils"
)

func setupDynamoTest(t *testing.T) (*DynamoConnector, string) {
	t.Helper()
	testutil.SkipIfServiceUnavailable(t, testutil.DynamoAddr, "DynamoDB")

	endpoint := "http://" + testutil.DynamoAddr
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithRegion("us-east-1"),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	client := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	// Ensure table exists (creates if missing, reuses if exists)
	tableName, err := testutil.EnsureDynamoTable(context.Background(), client)
	if err != nil {
		t.Fatalf("Failed to ensure DynamoDB table: %v", err)
	}

	// Set foundation_DATABASE_TABLE for the connector to use
	os.Setenv("foundation_DATABASE_TABLE", tableName)

	connector := NewDynamoConnector(client, nil)
	prefix := testutil.TestPrefix(t)

	return connector, prefix
}

func createTestUser(t *testing.T, prefix, suffix string) *models.User {
	t.Helper()
	id := prefix + suffix
	return models.NewUser(&id, authentication.AuthenticationProvider_Clerk, "auth_"+id, id+"@test.com", "Test User "+suffix, models.UserRole_User)
}

func TestDynamo_UpsertGet(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	user := createTestUser(t, prefix, "user1")

	// Upsert user
	err := connector.Upsert(ctx, nil, user)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Get user
	key := models.UserKeys.Key(user.UserID)
	result, err := connector.Get(ctx, nil, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result == nil {
		t.Fatal("Get returned nil")
	}

	gotUser, ok := result.(*models.User)
	if !ok {
		t.Fatalf("Get returned wrong type: %T", result)
	}

	if gotUser.UserID != user.UserID {
		t.Errorf("UserID = %q, want %q", gotUser.UserID, user.UserID)
	}
	if gotUser.Email != user.Email {
		t.Errorf("Email = %q, want %q", gotUser.Email, user.Email)
	}

	// Cleanup
	t.Cleanup(func() {
		connector.Delete(ctx, nil, key)
	})
}

func TestDynamo_GetNonExistent(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	key := models.UserKeys.Key(prefix + "nonexistent")

	result, err := connector.Get(ctx, nil, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil for non-existent item, got %v", result)
	}
}

func TestDynamo_Delete(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	user := createTestUser(t, prefix, "deleteuser")
	key := models.UserKeys.Key(user.UserID)

	// Upsert user
	err := connector.Upsert(ctx, nil, user)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify user exists
	result, err := connector.Get(ctx, nil, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result == nil {
		t.Fatal("User should exist after Upsert")
	}

	// Delete user
	err = connector.Delete(ctx, nil, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify user is gone
	result, err = connector.Get(ctx, nil, key)
	if err != nil {
		t.Fatalf("Get after Delete failed: %v", err)
	}
	if result != nil {
		t.Error("User should not exist after Delete")
	}
}

func TestDynamo_UpdateExisting(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	user := createTestUser(t, prefix, "updateuser")
	key := models.UserKeys.Key(user.UserID)

	// Upsert initial user
	err := connector.Upsert(ctx, nil, user)
	if err != nil {
		t.Fatalf("Initial Upsert failed: %v", err)
	}

	// Update user
	user.DisplayName = "Updated Name"
	user.Email = "updated@test.com"
	err = connector.Upsert(ctx, nil, user)
	if err != nil {
		t.Fatalf("Update Upsert failed: %v", err)
	}

	// Verify update
	result, err := connector.Get(ctx, nil, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	gotUser := result.(*models.User)
	if gotUser.DisplayName != "Updated Name" {
		t.Errorf("DisplayName = %q, want %q", gotUser.DisplayName, "Updated Name")
	}
	if gotUser.Email != "updated@test.com" {
		t.Errorf("Email = %q, want %q", gotUser.Email, "updated@test.com")
	}

	// Cleanup
	t.Cleanup(func() {
		connector.Delete(ctx, nil, key)
	})
}

func TestDynamo_BulkUpsert(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	users := []models.Model{
		createTestUser(t, prefix, "bulk1"),
		createTestUser(t, prefix, "bulk2"),
		createTestUser(t, prefix, "bulk3"),
	}

	// Bulk upsert
	err := connector.BulkUpsert(ctx, nil, users, nil)
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}

	// Verify each user exists
	for _, u := range users {
		user := u.(*models.User)
		key := models.UserKeys.Key(user.UserID)
		result, err := connector.Get(ctx, nil, key)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", user.UserID, err)
		}
		if result == nil {
			t.Errorf("User %s should exist after BulkUpsert", user.UserID)
		}
	}

	// Cleanup
	t.Cleanup(func() {
		for _, u := range users {
			user := u.(*models.User)
			key := models.UserKeys.Key(user.UserID)
			connector.Delete(ctx, nil, key)
		}
	})
}

func TestDynamo_BulkGet(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	users := []models.Model{
		createTestUser(t, prefix, "bulkget1"),
		createTestUser(t, prefix, "bulkget2"),
		createTestUser(t, prefix, "bulkget3"),
	}

	// Insert users first
	err := connector.BulkUpsert(ctx, nil, users, nil)
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}

	// Build keys
	keys := make([]dbSchema.Key, len(users))
	for i, u := range users {
		user := u.(*models.User)
		keys[i] = models.UserKeys.Key(user.UserID)
	}

	// Bulk get
	results, err := connector.BulkGet(ctx, nil, keys)
	if err != nil {
		t.Fatalf("BulkGet failed: %v", err)
	}

	if len(results) != len(users) {
		t.Errorf("BulkGet returned %d items, want %d", len(results), len(users))
	}

	// Cleanup
	t.Cleanup(func() {
		for _, u := range users {
			user := u.(*models.User)
			key := models.UserKeys.Key(user.UserID)
			connector.Delete(ctx, nil, key)
		}
	})
}

func TestDynamo_BulkDelete(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	users := []models.Model{
		createTestUser(t, prefix, "bulkdel1"),
		createTestUser(t, prefix, "bulkdel2"),
		createTestUser(t, prefix, "bulkdel3"),
	}

	// Insert users first
	err := connector.BulkUpsert(ctx, nil, users, nil)
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}

	// Build keys
	keys := make([]dbSchema.Key, len(users))
	for i, u := range users {
		user := u.(*models.User)
		keys[i] = models.UserKeys.Key(user.UserID)
	}

	// Bulk delete
	err = connector.BulkDelete(ctx, nil, keys, nil)
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}

	// Verify all deleted
	for i, key := range keys {
		result, err := connector.Get(ctx, nil, key)
		if err != nil {
			t.Fatalf("Get failed for key %d: %v", i, err)
		}
		if result != nil {
			t.Errorf("User %d should not exist after BulkDelete", i)
		}
	}
}

func TestDynamo_BulkUpsert_Atomic(t *testing.T) {
	connector, prefix := setupDynamoTest(t)
	ctx := context.Background()

	users := []models.Model{
		createTestUser(t, prefix, "atomic1"),
		createTestUser(t, prefix, "atomic2"),
		createTestUser(t, prefix, "atomic3"),
	}

	// Atomic bulk upsert
	opts := &dbSchema.BulkOptions{Atomic: true}
	err := connector.BulkUpsert(ctx, nil, users, opts)
	if err != nil {
		t.Fatalf("Atomic BulkUpsert failed: %v", err)
	}

	// Verify each user exists
	for _, u := range users {
		user := u.(*models.User)
		key := models.UserKeys.Key(user.UserID)
		result, err := connector.Get(ctx, nil, key)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", user.UserID, err)
		}
		if result == nil {
			t.Errorf("User %s should exist after atomic BulkUpsert", user.UserID)
		}
	}

	// Cleanup
	t.Cleanup(func() {
		keys := make([]dbSchema.Key, len(users))
		for i, u := range users {
			user := u.(*models.User)
			keys[i] = models.UserKeys.Key(user.UserID)
		}
		connector.BulkDelete(ctx, nil, keys, nil)
	})
}

// Ensure utils import is used
var _ = utils.GenerateULID
