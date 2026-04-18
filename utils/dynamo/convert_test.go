package dynamo

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/authentication"
	"github.com/cannonball10/foundation/schemas/database"
)

func TestKeyToAttributeValue_Valid(t *testing.T) {
	key := database.Key{"PK": "USER#1", "SK": "USER#1"}
	av, err := KeyToAttributeValue(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if av == nil {
		t.Fatal("expected non-nil result")
	}
	// Verify the PK value was marshaled correctly
	pk, ok := av["PK"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("expected PK to be string type, got %T", av["PK"])
	}
	if pk.Value != "USER#1" {
		t.Errorf("PK = %q, want %q", pk.Value, "USER#1")
	}
}

func TestKeyToAttributeValue_NilKey(t *testing.T) {
	_, err := KeyToAttributeValue(nil)
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}

func TestModelToAttributeValue_Valid(t *testing.T) {
	user := &models.User{
		UserID:                 "u1",
		AuthenticationProvider: authentication.AuthenticationProvider_Clerk,
		AuthenticationID:       "clerk_123",
		Email:                  "a@b.com",
	}

	av, err := ModelToAttributeValue(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check PK/SK were set
	pk, ok := av["PK"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("expected PK to be string type, got %T", av["PK"])
	}
	if pk.Value != "USER#u1" {
		t.Errorf("PK = %q, want %q", pk.Value, "USER#u1")
	}

	// Check GSI1 keys
	gsi1pk, ok := av["GSI1PK"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("expected GSI1PK to be string type, got %T", av["GSI1PK"])
	}
	if gsi1pk.Value != "PROVIDER#CLERK" {
		t.Errorf("GSI1PK = %q, want %q", gsi1pk.Value, "PROVIDER#CLERK")
	}
}

func TestModelToAttributeValue_NilModel(t *testing.T) {
	_, err := ModelToAttributeValue(nil)
	if err == nil {
		t.Fatal("expected error for nil model")
	}
}

func TestAttributeValueToModel_RoundTrip(t *testing.T) {
	user := &models.User{
		UserID:                 "u1",
		AuthenticationProvider: authentication.AuthenticationProvider_Clerk,
		AuthenticationID:       "clerk_123",
		Email:                  "a@b.com",
		DisplayName:            "Alice",
		Role:                   models.UserRole_User,
	}

	// Model → AV
	av, err := ModelToAttributeValue(user)
	if err != nil {
		t.Fatalf("ModelToAttributeValue: %v", err)
	}

	// AV → Model
	result, err := AttributeValueToModel(av)
	if err != nil {
		t.Fatalf("AttributeValueToModel: %v", err)
	}

	got, ok := result.(*models.User)
	if !ok {
		t.Fatalf("expected *models.User, got %T", result)
	}
	if got.Email != "a@b.com" {
		t.Errorf("Email = %q, want %q", got.Email, "a@b.com")
	}
}

func TestAttributeValueToModel_NilAV(t *testing.T) {
	_, err := AttributeValueToModel(nil)
	if err == nil {
		t.Fatal("expected error for nil attribute value map")
	}
}

func TestAttributeValueToKey_Valid(t *testing.T) {
	av := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#1"},
		"SK": &types.AttributeValueMemberS{Value: "USER#1"},
	}
	key, err := AttributeValueToKey(av)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key["PK"] != "USER#1" {
		t.Errorf("PK = %q, want %q", key["PK"], "USER#1")
	}
}

func TestAttributeValueToKey_Nil(t *testing.T) {
	_, err := AttributeValueToKey(nil)
	if err == nil {
		t.Fatal("expected error for nil AV")
	}
}

func TestModelToAttributeValue_NonListable(t *testing.T) {
	user := &models.User{
		UserID:                 "u1",
		AuthenticationProvider: authentication.AuthenticationProvider_Clerk,
		AuthenticationID:       "clerk_123",
		Email:                  "a@b.com",
	}

	av, err := ModelToAttributeValue(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// User does not implement Listable, so LISTPK/LISTSK should not be set
	if _, ok := av["LISTPK"]; ok {
		t.Error("expected LISTPK to not be set for non-listable model")
	}
	if _, ok := av["LISTSK"]; ok {
		t.Error("expected LISTSK to not be set for non-listable model")
	}
}

// init import to ensure User model is registered via its init()
var _ = attributevalue.Marshal
