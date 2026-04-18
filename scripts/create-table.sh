#!/bin/bash

# Script to create the DynamoDB table for local development
# Usage: ./create-table.sh

ENDPOINT="${DYNAMODB_ENDPOINT:-http://localhost:8000}"
TABLE_NAME="${DYNAMODB_TABLE_NAME:-dev}"

echo "Creating table '$TABLE_NAME' at $ENDPOINT..."

aws dynamodb create-table \
  --endpoint-url "$ENDPOINT" \
  --table-name "$TABLE_NAME" \
  --attribute-definitions \
    AttributeName=PK,AttributeType=S \
    AttributeName=SK,AttributeType=S \
    AttributeName=GSI1PK,AttributeType=S \
    AttributeName=GSI1SK,AttributeType=S \
    AttributeName=GSI2PK,AttributeType=S \
    AttributeName=GSI2SK,AttributeType=S \
    AttributeName=LISTPK,AttributeType=S \
    AttributeName=LISTSK,AttributeType=S \
  --key-schema \
    AttributeName=PK,KeyType=HASH \
    AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --global-secondary-indexes \
    'IndexName=GSI1,KeySchema=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL}' \
    'IndexName=GSI2,KeySchema=[{AttributeName=GSI2PK,KeyType=HASH},{AttributeName=GSI2SK,KeyType=RANGE}],Projection={ProjectionType=ALL}' \
    'IndexName=LIST,KeySchema=[{AttributeName=LISTPK,KeyType=HASH},{AttributeName=LISTSK,KeyType=RANGE}],Projection={ProjectionType=ALL}'

echo ""
echo "Waiting for table to be active..."
aws dynamodb wait table-exists \
  --endpoint-url "$ENDPOINT" \
  --table-name "$TABLE_NAME"

echo "Table '$TABLE_NAME' created successfully!"
