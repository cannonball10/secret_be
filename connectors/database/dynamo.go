package database

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	databaseErrors "github.com/cannonball10/foundation/errors/database"
	"github.com/cannonball10/foundation/models"
	dbSchema "github.com/cannonball10/foundation/schemas/database"
	dynamoUtils "github.com/cannonball10/foundation/utils/dynamo"
)

const (
	defaultBatchGetLimit   = 100
	defaultBatchWriteLimit = 25
	defaultMaxRetries      = 5
	defaultRetryDelay      = 100 * time.Millisecond
	transactWriteLimit     = 25 // DynamoDB hard limit for TransactWriteItems
)

// DynamoConnectorConfig controls batch behavior and retry policy.
type DynamoConnectorConfig struct {
	BatchGetLimit   int
	BatchWriteLimit int
	MaxRetries      int
	RetryDelay      time.Duration
}

// DynamoConnector implements DatabaseConnector using AWS SDK v2 DynamoDB client.
type DynamoConnector struct {
	client        *dynamodb.Client
	clientOnce    sync.Once
	clientFactory func() *dynamodb.Client

	batchGetLimit   int
	batchWriteLimit int
	maxRetries      int
	retryDelay      time.Duration
}

// NewDynamoConnector returns a connector with an eagerly provided client.
func NewDynamoConnector(client *dynamodb.Client, cfg *DynamoConnectorConfig) *DynamoConnector {
	return newDynamoConnector(client, nil, cfg)
}

// NewDynamoConnectorLazy returns a connector that initializes the client on first use.
func NewDynamoConnectorLazy(factory func() *dynamodb.Client, cfg *DynamoConnectorConfig) *DynamoConnector {
	return newDynamoConnector(nil, factory, cfg)
}

// DefaultDynamoConnector loads AWS default config and creates a connector.
func DefaultDynamoDatabseConnector(ctx context.Context, cfg *DynamoConnectorConfig) (*DynamoConnector, error) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	// AWS config: local DynamoDB if endpoint set, else default credential chain
	var awsCfg aws.Config
	var err error
	if endpoint != "" {
		slog.Info("using local DynamoDB endpoint", "endpoint", endpoint)
		awsCfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("h7ehk8", "pt3tz", "")),
			config.WithRegion("us-east-1"),
			config.WithBaseEndpoint(endpoint),
		)
		if err != nil {
			return nil, err
		}
	} else {
		awsCfg, err = config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	var dynamoClient *dynamodb.Client
	if endpoint != "" {
		dynamoClient = dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	} else {
		dynamoClient = dynamodb.NewFromConfig(awsCfg)
	}

	return NewDynamoConnector(dynamoClient, cfg), nil
}

func newDynamoConnector(client *dynamodb.Client, factory func() *dynamodb.Client, cfg *DynamoConnectorConfig) *DynamoConnector {
	limits := applyDynamoDefaults(cfg)
	return &DynamoConnector{
		client:          client,
		clientFactory:   factory,
		batchGetLimit:   limits.BatchGetLimit,
		batchWriteLimit: limits.BatchWriteLimit,
		maxRetries:      limits.MaxRetries,
		retryDelay:      limits.RetryDelay,
	}
}

func applyDynamoDefaults(cfg *DynamoConnectorConfig) DynamoConnectorConfig {
	if cfg == nil {
		return DynamoConnectorConfig{
			BatchGetLimit:   defaultBatchGetLimit,
			BatchWriteLimit: defaultBatchWriteLimit,
			MaxRetries:      defaultMaxRetries,
			RetryDelay:      defaultRetryDelay,
		}
	}

	limits := *cfg
	if limits.BatchGetLimit <= 0 {
		limits.BatchGetLimit = defaultBatchGetLimit
	}
	if limits.BatchWriteLimit <= 0 {
		limits.BatchWriteLimit = defaultBatchWriteLimit
	}
	if limits.MaxRetries <= 0 {
		limits.MaxRetries = defaultMaxRetries
	}
	if limits.RetryDelay <= 0 {
		limits.RetryDelay = defaultRetryDelay
	}
	return limits
}

func (c *DynamoConnector) Get(ctx context.Context, collection *string, key dbSchema.Key) (models.Model, error) {
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return nil, err
	}

	keyAV, err := dynamoUtils.KeyToAttributeValue(key)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &table,
		Key:       keyAV,
	})
	if err != nil {
		slog.ErrorContext(ctx, "dynamodb GetItem failed", "table", table, "error", err)
		return nil, err
	}

	if resp.Item == nil {
		return nil, nil
	}

	return dynamoUtils.AttributeValueToModel(resp.Item)
}

func (c *DynamoConnector) Query(ctx context.Context, collection *string, qi dbSchema.QueryInput, opts dbSchema.QueryOptions) (*dbSchema.QueryOutput, error) {
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return nil, err
	}

	// Determine PK/SK attribute names based on index
	pkAttr, skAttr := "PK", "SK"
	if qi.IndexName != nil {
		pkAttr = string(*qi.IndexName) + "PK"
		skAttr = string(*qi.IndexName) + "SK"
	}

	// Build expression attribute names and values
	exprNames := map[string]string{
		"#pk": pkAttr,
	}
	exprValues := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: qi.PartitionKey},
	}

	keyCondition := "#pk = :pk"

	// Append sort key condition if provided
	if sk := qi.SortKey; sk != nil {
		exprNames["#sk"] = skAttr

		switch {
		case sk.EQ != nil:
			exprValues[":sk"] = &types.AttributeValueMemberS{Value: *sk.EQ}
			keyCondition += " AND #sk = :sk"
		case sk.BeginsWith != nil:
			exprValues[":sk"] = &types.AttributeValueMemberS{Value: *sk.BeginsWith}
			keyCondition += " AND begins_with(#sk, :sk)"
		case sk.Between != nil:
			exprValues[":sklo"] = &types.AttributeValueMemberS{Value: sk.Between[0]}
			exprValues[":skhi"] = &types.AttributeValueMemberS{Value: sk.Between[1]}
			keyCondition += " AND #sk BETWEEN :sklo AND :skhi"
		case sk.LT != nil:
			exprValues[":sk"] = &types.AttributeValueMemberS{Value: *sk.LT}
			keyCondition += " AND #sk < :sk"
		case sk.LE != nil:
			exprValues[":sk"] = &types.AttributeValueMemberS{Value: *sk.LE}
			keyCondition += " AND #sk <= :sk"
		case sk.GT != nil:
			exprValues[":sk"] = &types.AttributeValueMemberS{Value: *sk.GT}
			keyCondition += " AND #sk > :sk"
		case sk.GE != nil:
			exprValues[":sk"] = &types.AttributeValueMemberS{Value: *sk.GE}
			keyCondition += " AND #sk >= :sk"
		}
	}

	input := &dynamodb.QueryInput{
		TableName:                 &table,
		KeyConditionExpression:    &keyCondition,
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
	}

	if qi.IndexName != nil {
		input.IndexName = aws.String(string(*qi.IndexName))
	}

	// Apply limit
	if opts.Limit > 0 {
		limit := int32(opts.Limit)
		input.Limit = &limit
	}

	if opts.Cursor != nil {
		cursorAV, err := attributevalue.MarshalMap(opts.Cursor)
		if err != nil {
			return nil, err
		}
		input.ExclusiveStartKey = cursorAV
	}

	if opts.Direction != "" {
		forward := opts.Direction == dbSchema.SortAscending
		input.ScanIndexForward = &forward
	}

	resp, err := client.Query(ctx, input)
	if err != nil {
		slog.ErrorContext(ctx, "dynamodb Query failed", "table", table, "pk", qi.PartitionKey, "error", err)
		return nil, err
	}

	results := make([]any, 0, len(resp.Items))
	for _, item := range resp.Items {
		model, err := dynamoUtils.AttributeValueToModel(item)
		if err != nil {
			return nil, err
		}
		results = append(results, model)
	}

	var page *dbSchema.QueryPage
	if resp.LastEvaluatedKey != nil {
		var nextCursor map[string]string
		if err := attributevalue.UnmarshalMap(resp.LastEvaluatedKey, &nextCursor); err != nil {
			return nil, err
		}
		page = &dbSchema.QueryPage{NextCursor: nextCursor}
	}

	return &dbSchema.QueryOutput{Models: results, Page: page}, nil
}

func (c *DynamoConnector) Upsert(ctx context.Context, collection *string, item models.Model) error {
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return err
	}

	itemAV, err := dynamoUtils.ModelToAttributeValue(item)
	if err != nil {
		return err
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &table,
		Item:      itemAV,
	})
	if err != nil {
		slog.ErrorContext(ctx, "dynamodb PutItem failed", "table", table, "error", err)
	}
	return err
}

func (c *DynamoConnector) Delete(ctx context.Context, collection *string, key dbSchema.Key) error {
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return err
	}

	keyAV, err := dynamoUtils.KeyToAttributeValue(key)
	if err != nil {
		return err
	}

	_, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &table,
		Key:       keyAV,
	})
	if err != nil {
		slog.ErrorContext(ctx, "dynamodb DeleteItem failed", "table", table, "error", err)
	}
	return err
}

func (c *DynamoConnector) BulkGet(ctx context.Context, collection *string, keys []dbSchema.Key) ([]models.Model, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return nil, err
	}

	var items []models.Model
	for _, chunk := range dynamoUtils.ChunkSlice(keys, c.batchGetLimit) {
		chunkItems, err := c.batchGetWithRetry(ctx, client, table, chunk)
		if err != nil {
			return nil, err
		}
		items = append(items, chunkItems...)
	}
	return items, nil
}

func (c *DynamoConnector) BulkUpsert(ctx context.Context, collection *string, items []models.Model, opts *dbSchema.BulkOptions) error {
	if len(items) == 0 {
		return nil
	}
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return err
	}

	useAtomic := opts != nil && opts.Atomic
	chunkSize := c.batchWriteLimit
	if useAtomic {
		chunkSize = transactWriteLimit
	}

	for _, chunk := range dynamoUtils.ChunkSlice(items, chunkSize) {
		if useAtomic {
			if err := c.transactWriteWithRetry(ctx, client, table, chunk, nil); err != nil {
				return err
			}
		} else {
			if err := c.batchWriteWithRetry(ctx, client, table, chunk, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *DynamoConnector) BulkDelete(ctx context.Context, collection *string, keys []dbSchema.Key, opts *dbSchema.BulkOptions) error {
	if len(keys) == 0 {
		return nil
	}
	client, table, err := c.getClientAndTable(collection)
	if err != nil {
		return err
	}

	useAtomic := opts != nil && opts.Atomic
	chunkSize := c.batchWriteLimit
	if useAtomic {
		chunkSize = transactWriteLimit
	}

	for _, chunk := range dynamoUtils.ChunkSlice(keys, chunkSize) {
		if useAtomic {
			if err := c.transactWriteWithRetry(ctx, client, table, nil, chunk); err != nil {
				return err
			}
		} else {
			if err := c.batchWriteWithRetry(ctx, client, table, nil, chunk); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *DynamoConnector) ListCollections(ctx context.Context) ([]string, error) {
	// For single-table logical collections, return empty list
	// since we're not persisting a metadata registry
	return []string{}, nil
}

func (c *DynamoConnector) CreateCollection(ctx context.Context, collection *string) error {
	// For single-table logical collections, just validate the collection name
	if collection == nil || *collection == "" {
		return databaseErrors.ErrDatabaseCollectionMissing
	}
	// No-op: collection is just a logical namespace in the single table
	return nil
}

func (c *DynamoConnector) DeleteCollection(ctx context.Context, collection *string) error {
	// For single-table logical collections, just validate the collection name
	if collection == nil || *collection == "" {
		return databaseErrors.ErrDatabaseCollectionMissing
	}
	// No-op: collection is just a logical namespace in the single table
	return nil
}

// getClientAndTable is a helper that gets the client and resolves the table name.
func (c *DynamoConnector) getClientAndTable(collection *string) (*dynamodb.Client, string, error) {
	client, err := c.getClient()
	if err != nil {
		return nil, "", err
	}
	table, err := dynamoUtils.ResolveCollection(collection)
	if err != nil {
		return nil, "", err
	}
	return client, table, nil
}

func (c *DynamoConnector) getClient() (*dynamodb.Client, error) {
	c.clientOnce.Do(func() {
		if c.client == nil && c.clientFactory != nil {
			c.client = c.clientFactory()
		}
	})
	if c.client == nil {
		return nil, databaseErrors.ErrDatabaseClientUninitialized
	}
	return c.client, nil
}

// Close releases any resources held by the DynamoDB client.
func (c *DynamoConnector) Close() error {
	// AWS SDK v2 DynamoDB client doesn't have an explicit Close method,
	// but we nil out the reference to allow garbage collection.
	c.client = nil
	return nil
}

// batchGetWithRetry performs a batch get operation with retry logic for unprocessed keys.
func (c *DynamoConnector) batchGetWithRetry(ctx context.Context, client *dynamodb.Client, collection string, keys []dbSchema.Key) ([]models.Model, error) {
	var results []models.Model
	remaining := keys

	for attempt := 0; attempt <= c.maxRetries && len(remaining) > 0; attempt++ {
		// Build request keys
		requestKeys := make([]map[string]types.AttributeValue, 0, len(remaining))
		for _, key := range remaining {
			keyAV, err := dynamoUtils.KeyToAttributeValue(key)
			if err != nil {
				return nil, err
			}
			requestKeys = append(requestKeys, keyAV)
		}

		resp, err := client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
			RequestItems: map[string]types.KeysAndAttributes{
				collection: {
					Keys: requestKeys,
				},
			},
		})
		if err != nil {
			slog.ErrorContext(ctx, "dynamodb BatchGetItem failed", "table", collection, "key_count", len(requestKeys), "attempt", attempt, "error", err)
			return nil, err
		}

		// Process responses
		tableResponses, ok := resp.Responses[collection]
		if ok {
			for _, item := range tableResponses {
				model, err := dynamoUtils.AttributeValueToModel(item)
				if err != nil {
					return nil, err
				}
				results = append(results, model)
			}
		}

		// Handle unprocessed keys
		unprocessed, ok := resp.UnprocessedKeys[collection]
		if ok && len(unprocessed.Keys) > 0 {
			slog.WarnContext(ctx, "dynamodb BatchGetItem has unprocessed keys", "table", collection, "unprocessed_count", len(unprocessed.Keys), "attempt", attempt)
			remaining = make([]dbSchema.Key, 0, len(unprocessed.Keys))
			for _, keyAV := range unprocessed.Keys {
				key, err := dynamoUtils.AttributeValueToKey(keyAV)
				if err != nil {
					return nil, err
				}
				remaining = append(remaining, key)
			}
		} else {
			remaining = nil
		}

		if len(remaining) > 0 {
			if err := dynamoUtils.SleepWithContext(ctx, c.retryDelay); err != nil {
				return nil, err
			}
		}
	}

	if len(remaining) > 0 {
		return nil, databaseErrors.ErrDatabaseBatchGetExceededRetries
	}
	return results, nil
}

// batchWriteWithRetry performs a batch write operation with retry logic for unprocessed items.
func (c *DynamoConnector) batchWriteWithRetry(ctx context.Context, client *dynamodb.Client, collection string, puts []models.Model, deletes []dbSchema.Key) error {
	remainingPuts := puts
	remainingDeletes := deletes

	for attempt := 0; attempt <= c.maxRetries && (len(remainingPuts) > 0 || len(remainingDeletes) > 0); attempt++ {
		writeRequests := make([]types.WriteRequest, 0, len(remainingPuts)+len(remainingDeletes))

		// Add put requests
		for _, model := range remainingPuts {
			itemAV, err := dynamoUtils.ModelToAttributeValue(model)
			if err != nil {
				return err
			}
			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: itemAV,
				},
			})
		}

		// Add delete requests
		for _, key := range remainingDeletes {
			keyAV, err := dynamoUtils.KeyToAttributeValue(key)
			if err != nil {
				return err
			}
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: keyAV,
				},
			})
		}

		resp, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				collection: writeRequests,
			},
		})
		if err != nil {
			slog.ErrorContext(ctx, "dynamodb BatchWriteItem failed", "table", collection, "key_count", len(writeRequests), "attempt", attempt, "error", err)
			return err
		}

		// Handle unprocessed items
		unprocessed, ok := resp.UnprocessedItems[collection]
		if ok && len(unprocessed) > 0 {
			slog.WarnContext(ctx, "dynamodb BatchWriteItem has unprocessed items", "table", collection, "unprocessed_count", len(unprocessed), "attempt", attempt)
			remainingPuts = make([]models.Model, 0)
			remainingDeletes = make([]dbSchema.Key, 0)
			for _, req := range unprocessed {
				if req.PutRequest != nil {
					model, err := dynamoUtils.AttributeValueToModel(req.PutRequest.Item)
					if err != nil {
						return err
					}
					remainingPuts = append(remainingPuts, model)
				}
				if req.DeleteRequest != nil {
					key, err := dynamoUtils.AttributeValueToKey(req.DeleteRequest.Key)
					if err != nil {
						return err
					}
					remainingDeletes = append(remainingDeletes, key)
				}
			}
		} else {
			remainingPuts = nil
			remainingDeletes = nil
		}

		if len(remainingPuts) > 0 || len(remainingDeletes) > 0 {
			if err := dynamoUtils.SleepWithContext(ctx, c.retryDelay); err != nil {
				return err
			}
		}
	}

	if len(remainingPuts) > 0 || len(remainingDeletes) > 0 {
		return databaseErrors.ErrDatabaseBatchWriteExceededRetries
	}
	return nil
}

// transactWriteWithRetry performs a transaction write operation with retry logic.
func (c *DynamoConnector) transactWriteWithRetry(ctx context.Context, client *dynamodb.Client, table string, puts []models.Model, deletes []dbSchema.Key) error {
	// DynamoDB transactions have a hard limit of 25 items
	if len(puts)+len(deletes) > transactWriteLimit {
		return databaseErrors.ErrDatabaseTransactionWriteExceededLimit
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		transactItems := make([]types.TransactWriteItem, 0, len(puts)+len(deletes))

		// Add put requests
		for _, model := range puts {
			itemAV, err := dynamoUtils.ModelToAttributeValue(model)
			if err != nil {
				return err
			}
			transactItems = append(transactItems, types.TransactWriteItem{
				Put: &types.Put{
					TableName: &table,
					Item:      itemAV,
				},
			})
		}

		// Add delete requests
		for _, key := range deletes {
			keyAV, err := dynamoUtils.KeyToAttributeValue(key)
			if err != nil {
				return err
			}
			transactItems = append(transactItems, types.TransactWriteItem{
				Delete: &types.Delete{
					TableName: &table,
					Key:       keyAV,
				},
			})
		}

		_, err := client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: transactItems,
		})

		if err == nil {
			return nil
		}

		slog.ErrorContext(ctx, "dynamodb TransactWriteItems failed", "table", table, "item_count", len(transactItems), "attempt", attempt, "error", err)

		// Check if error is retryable (transient errors)
		if !isRetryableError(err) {
			return err
		}

		// Retry with exponential backoff
		if attempt < c.maxRetries {
			delay := c.retryDelay * time.Duration(1<<uint(attempt))
			if err := dynamoUtils.SleepWithContext(ctx, delay); err != nil {
				return err
			}
		}
	}

	return databaseErrors.ErrDatabaseTransactionWriteExceededRetries
}

// isRetryableError determines if an error is retryable for transaction operations.
func isRetryableError(err error) bool {
	var te *types.TransactionCanceledException
	if errors.As(err, &te) {
		// TransactionCanceledException can be retried if it's due to transient issues
		// Check if it's a conditional check failure (not retryable) or other issues
		for _, reason := range te.CancellationReasons {
			if reason.Code != nil && *reason.Code == "None" {
				// This item succeeded, but others may have failed
				continue
			}
			// If it's a conditional check failure, it's not retryable
			if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
				return false
			}
			// Other cancellation reasons might be retryable (e.g., throttling)
			return true
		}
		return false
	}

	// For other errors, check if they're retryable (e.g., throttling, service errors)
	var tce *types.ProvisionedThroughputExceededException
	var rce *types.RequestLimitExceeded
	var ie *types.InternalServerError
	return errors.As(err, &tce) || errors.As(err, &rce) || errors.As(err, &ie)
}
