//go:build integration

package testutil

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	// DefaultDynamoTable is the default table name when foundation_DATABASE_TABLE is not set
	DefaultDynamoTable = "dev"
)

// DynamoTableSchema defines the standard table schema matching models/dynamo.go
var DynamoTableSchema = &dynamodb.CreateTableInput{
	AttributeDefinitions: []types.AttributeDefinition{
		{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
		{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
		{AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS},
		{AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS},
	},
	KeySchema: []types.KeySchemaElement{
		{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
		{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
	},
	GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
		{
			IndexName: aws.String("GSI1"),
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String("GSI1PK"), KeyType: types.KeyTypeHash},
				{AttributeName: aws.String("GSI1SK"), KeyType: types.KeyTypeRange},
			},
			Projection: &types.Projection{
				ProjectionType: types.ProjectionTypeAll,
			},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5),
				WriteCapacityUnits: aws.Int64(5),
			},
		},
	},
	ProvisionedThroughput: &types.ProvisionedThroughput{
		ReadCapacityUnits:  aws.Int64(5),
		WriteCapacityUnits: aws.Int64(5),
	},
}

// GetDynamoTableName returns the table name from foundation_DATABASE_TABLE env var,
// defaulting to "dev" if not set.
func GetDynamoTableName() string {
	tableName := os.Getenv("foundation_DATABASE_TABLE")
	if tableName == "" {
		return DefaultDynamoTable
	}
	return tableName
}

// EnsureDynamoTable checks if the table exists, creating it if necessary.
// Returns the table name used.
func EnsureDynamoTable(ctx context.Context, client *dynamodb.Client) (string, error) {
	tableName := GetDynamoTableName()

	// Check if table exists
	exists, err := dynamoTableExists(ctx, client, tableName)
	if err != nil {
		return "", err
	}

	if exists {
		return tableName, nil
	}

	// Create table using the schema
	input := *DynamoTableSchema
	input.TableName = aws.String(tableName)

	_, err = client.CreateTable(ctx, &input)
	if err != nil {
		return "", err
	}

	// Wait for table to be active
	waiter := dynamodb.NewTableExistsWaiter(client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}, 60000000000) // 60 seconds
	if err != nil {
		return "", err
	}

	return tableName, nil
}

// dynamoTableExists checks if a DynamoDB table exists.
func dynamoTableExists(ctx context.Context, client *dynamodb.Client, tableName string) (bool, error) {
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
