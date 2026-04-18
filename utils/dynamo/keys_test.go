package dynamo

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestExtractPKSK_Valid(t *testing.T) {
	av := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#123"},
		"SK": &types.AttributeValueMemberS{Value: "USER#123"},
	}

	pk, sk, err := extractPKSK(av)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pk != "USER#123" {
		t.Errorf("PK = %q, want %q", pk, "USER#123")
	}
	if sk != "USER#123" {
		t.Errorf("SK = %q, want %q", sk, "USER#123")
	}
}

func TestExtractPKSK_MissingPK(t *testing.T) {
	av := map[string]types.AttributeValue{
		"SK": &types.AttributeValueMemberS{Value: "USER#123"},
	}

	_, _, err := extractPKSK(av)
	if err == nil {
		t.Fatal("expected error for missing PK")
	}
}

func TestExtractPKSK_MissingSK(t *testing.T) {
	av := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#123"},
	}

	_, _, err := extractPKSK(av)
	if err == nil {
		t.Fatal("expected error for missing SK")
	}
}

func TestExtractPKSK_NilPK(t *testing.T) {
	av := map[string]types.AttributeValue{
		"PK": nil,
		"SK": &types.AttributeValueMemberS{Value: "USER#123"},
	}

	_, _, err := extractPKSK(av)
	if err == nil {
		t.Fatal("expected error for nil PK")
	}
}

func TestExtractPKSK_NilSK(t *testing.T) {
	av := map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: "USER#123"},
		"SK": nil,
	}

	_, _, err := extractPKSK(av)
	if err == nil {
		t.Fatal("expected error for nil SK")
	}
}
