# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run single test
go test -run TestName ./path/to/package

# Format code
gofmt -w .

# Static analysis
go vet ./...

# Start local services (Redis, DynamoDB, Neo4j, Qdrant)
docker-compose up -d

# Create DynamoDB table locally
./scripts/create-table.sh
```

## Architecture

This is a Go backend with a **connector-based architecture** for integrating external services. All integrations follow a consistent interface pattern enabling dependency injection and testability.

### Core Patterns

**Connectors** (`connectors/`): Interface-based adapters for external services. Each connector type has:
- An interface definition (e.g., `DatabaseConnector`)
- A default implementation (e.g., `DefaultDatabaseConnector`)
- Optional lifecycle hooks (`Init()`, `Close()`)

**Dependencies** (`dependencies/`): Assembles all connectors and handlers into a single `Dependencies` struct that embeds both `Connectors` and `Handlers`. This is the main DI container passed through the application.

**Models** (`models/`): DynamoDB-backed data models implementing the `Model` interface. Models define their own key structure (PK/SK/GSI) and are auto-registered with a global registry via `models.Register()`. The registry enables `models.Lookup(pk, sk)` to instantiate the correct model type from raw DynamoDB items.

**Handlers** (`handlers/`): Business logic that orchestrates connectors. Handlers receive the `Connectors` struct and expose domain operations.

**Schemas** (`schemas/`): Data contracts and constants shared across packages (query filters, relationship types, storage options).

### Connector Types

- `authentication/` - Clerk-based auth (token verification, user creation)
- `cache/` - Redis and in-memory caching
- `channel/` - SMS via Twilio
- `database/` - DynamoDB operations (Get, Upsert, Query)
- `embedding/` - OpenAI text embeddings
- `feature/` - LaunchDarkly feature flags
- `graphdb/` - Neo4j graph relationships
- `notification/` - Slack messaging
- `secret/` - AWS Secrets Manager
- `storage/` - S3 file storage
- `vector/` - Qdrant vector search

### DynamoDB Model Pattern

Models use composite keys with prefixes for type identification:
```go
type User struct {
    models.DynamoMetadata
    ID    string
    Email string
}

func (u *User) TypeIdentifier() (string, string) {
    return "USER#", "USER#"
}
```

Register models at init time:
```go
func init() {
    models.Register("USER#", "USER#", func() models.Model { return &User{} })
}
```

### Local Development

Docker Compose provides:
- Redis (6379)
- DynamoDB Local (8000) with Admin UI (8001)
- Neo4j (7474/7687)
- Qdrant (6333/6334)

Environment variables are managed via `.envrc` (direnv).
