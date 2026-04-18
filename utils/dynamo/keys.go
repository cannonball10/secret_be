package dynamo

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// extractPKSK extracts PK and SK string values from a DynamoDB AttributeValue map.
func extractPKSK(av map[string]types.AttributeValue) (pk, sk string, err error) {
	pkAV, pkExists := av["PK"]
	if !pkExists {
		return "", "", errors.New("PK attribute is missing")
	}
	if pkAV == nil {
		return "", "", errors.New("PK attribute is nil")
	}

	skAV, skExists := av["SK"]
	if !skExists {
		return "", "", errors.New("SK attribute is missing")
	}
	if skAV == nil {
		return "", "", errors.New("SK attribute is nil")
	}

	// Extract string values from AttributeValue
	var pkStr string
	if err := attributevalue.Unmarshal(pkAV, &pkStr); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal PK: %w", err)
	}

	var skStr string
	if err := attributevalue.Unmarshal(skAV, &skStr); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal SK: %w", err)
	}

	return pkStr, skStr, nil
}
