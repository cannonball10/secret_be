package dynamo

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cannonball10/foundation/models"
	dbSchema "github.com/cannonball10/foundation/schemas/database"
)

// KeyToAttributeValue converts a database.Key to AWS SDK AttributeValue map.
func KeyToAttributeValue(key dbSchema.Key) (map[string]types.AttributeValue, error) {
	if key == nil {
		return nil, errors.New("key cannot be nil")
	}
	return attributevalue.MarshalMap(key)
}

// AttributeValueToKey converts an AWS SDK AttributeValue map to database.Key.
func AttributeValueToKey(av map[string]types.AttributeValue) (dbSchema.Key, error) {
	if av == nil {
		return nil, errors.New("attribute value map cannot be nil")
	}
	var key dbSchema.Key
	if err := attributevalue.UnmarshalMap(av, &key); err != nil {
		return nil, err
	}
	return key, nil
}

// ModelToAttributeValue converts a models.Model to AWS SDK AttributeValue map.
// It marshals the model's data fields, then overlays PK/SK and GSI keys.
func ModelToAttributeValue(model models.Model) (map[string]types.AttributeValue, error) {
	if model == nil {
		return nil, errors.New("model cannot be nil")
	}

	// Marshal the model (data fields only)
	itemAV, err := attributevalue.MarshalMap(model)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal model: %w", err)
	}

	// Set PK and SK from the model interface
	pkAV, err := attributevalue.Marshal(model.PK())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PK: %w", err)
	}
	skAV, err := attributevalue.Marshal(model.SK())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SK: %w", err)
	}
	itemAV["PK"] = pkAV
	itemAV["SK"] = skAV

	// Set GSI keys from the model interface
	for idx, gsi := range model.GSIs() {
		gpk, err := attributevalue.Marshal(gsi.PK)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal GSI%dPK: %w", idx, err)
		}
		gsk, err := attributevalue.Marshal(gsi.SK)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal GSI%dSK: %w", idx, err)
		}
		itemAV[fmt.Sprintf("GSI%dPK", idx)] = gpk
		itemAV[fmt.Sprintf("GSI%dSK", idx)] = gsk
	}

	// Set LIST GSI keys if the model supports listing
	if listable, ok := model.(models.Listable); ok {
		lpk, err := attributevalue.Marshal(listable.ListPK())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal LISTPK: %w", err)
		}
		lsk, err := attributevalue.Marshal(listable.ListSK())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal LISTSK: %w", err)
		}
		itemAV["LISTPK"] = lpk
		itemAV["LISTSK"] = lsk
	}

	return itemAV, nil
}

// AttributeValueToModel converts an AWS SDK AttributeValue map to a models.Model.
// It uses the model registry to determine the correct type from PK/SK prefixes.
func AttributeValueToModel(av map[string]types.AttributeValue) (models.Model, error) {
	if av == nil {
		return nil, errors.New("attribute value map cannot be nil")
	}

	// Extract PK and SK to determine model type
	pk, sk, err := extractPKSK(av)
	if err != nil {
		return nil, err
	}

	// Look up the correct model type from registry
	model, err := models.Lookup(pk, sk)
	if err != nil {
		return nil, err
	}

	// Unmarshal the AttributeValue map into the concrete model type
	if err := attributevalue.UnmarshalMap(av, model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into model: %w", err)
	}

	return model, nil
}
