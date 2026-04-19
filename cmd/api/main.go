// Command api is a minimal entrypoint for the Secret Hitler backend.
// It wires just the DynamoDB connector, picks an authenticator based
// on the environment, and starts the Gin HTTP server from package api.
//
// Intentionally narrow: importing the full `connectors` package pulls
// in LiveKit (which currently has unresolved dependencies), and the
// Secret Hitler engine only needs a DatabaseConnector. Add connectors
// as the game starts using them (Redis pub-sub for the hub, Clerk, …).
//
// Run:
//
//	cp .env.example .env
//	docker-compose up -d dynamodb
//	./scripts/create-table.sh
//	go run ./cmd/api
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/cannonball10/foundation/api"
	authconn "github.com/cannonball10/foundation/connectors/authentication"
	"github.com/cannonball10/foundation/connectors/database"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.Warn("could not read .env", "err", err)
	}

	ctx := context.Background()

	db, err := database.DefaultDynamoDatabseConnector(ctx, nil)
	if err != nil {
		slog.Error("database init failed", "err", err)
		os.Exit(1)
	}

	auth := pickAuth(ctx)

	server := api.NewServer(api.Options{
		Database: db,
		Auth:     auth,
		GinMode:  ginMode(),
	})

	addr := ":" + envOr("API_PORT", "8080")
	slog.Info("starting api", "addr", addr, "auth", authKind(auth))
	if err := server.Run(addr); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

// pickAuth returns ClerkAuth when CLERK_SECRET_KEY is set, else NopAuth.
func pickAuth(ctx context.Context) api.Authenticator {
	if os.Getenv("CLERK_SECRET_KEY") == "" {
		return api.NopAuth{}
	}
	conn, err := authconn.DefaultClerkAuthenticationConnector(ctx)
	if err != nil {
		slog.Warn("clerk init failed, falling back to NopAuth", "err", err)
		return api.NopAuth{}
	}
	return api.ClerkAuth{Connector: conn}
}

func authKind(a api.Authenticator) string {
	if _, ok := a.(api.NopAuth); ok {
		return "nop"
	}
	return "clerk"
}

func ginMode() string {
	if m := os.Getenv("GIN_MODE"); m != "" {
		return m
	}
	return gin.ReleaseMode
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
