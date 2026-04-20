// Command api is a minimal entrypoint for the Replicant backend.
// It wires just the DynamoDB connector, picks an authenticator based
// on the environment, and starts the Gin HTTP server from package api.
//
// Intentionally narrow: importing the full `connectors` package pulls
// in LiveKit (which currently has unresolved dependencies), and the
// Replicant engine only needs a DatabaseConnector. Add connectors
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
	"strconv"
	"strings"
	"time"

	"github.com/cannonball10/foundation/api"
	authconn "github.com/cannonball10/foundation/connectors/authentication"
	"github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/connectors/inference"
	"github.com/cannonball10/foundation/connectors/tts"
	"github.com/cannonball10/foundation/handlers/game"
	"github.com/cannonball10/foundation/handlers/narrator"
	"github.com/cannonball10/foundation/handlers/rulesbot"
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

	simulate, simCfg := simulateOpts()
	narr := buildNarrator(ctx)
	bot := buildRulesBot(ctx)
	server := api.NewServer(api.Options{
		Database:         db,
		Auth:             auth,
		GinMode:          ginMode(),
		SimulateOnCreate: simulate,
		SimulateConfig:   simCfg,
		AllowedOrigins:   parseCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		Narrator:         narr,
		RulesBot:         bot,
	})

	addr := ":" + envOr("API_PORT", "8080")
	slog.Info("starting api", "addr", addr, "auth", authKind(auth), "simulate", simulate)
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

// simulateOpts reads the SIMULATE_* env vars. When SIMULATE_ON_CREATE is
// truthy, every successful CreateGame fires a bot-driven playthrough.
//
//	SIMULATE_ON_CREATE   "1"|"true"|"yes"  — enable
//	SIMULATE_PLAYERS     int               — total seats incl. host (default 7, clamped to [5,10])
//	SIMULATE_STEP_MS     int milliseconds  — pause between bot actions (default 1500)
//	SIMULATE_START_MS    int milliseconds  — pause before first bot joins (default 2000)
//	SIMULATE_SEED        uint64            — seeds bot RNG for reproducible demos
//	SIMULATE_HUMAN_SEATS int               — seats reserved for real players; bots don't fill them
//	                                         and the simulator waits for the host to press Start
//	                                         (default 0 = fully autonomous demo)
func simulateOpts() (bool, game.SimulationConfig) {
	if !truthy(os.Getenv("SIMULATE_ON_CREATE")) {
		return false, game.SimulationConfig{}
	}
	return true, game.SimulationConfig{
		Players:    atoiOr("SIMULATE_PLAYERS", 0),
		StepDelay:  msEnv("SIMULATE_STEP_MS"),
		StartDelay: msEnv("SIMULATE_START_MS"),
		Seed:       uint64(atoiOr("SIMULATE_SEED", 0)),
		HumanSeats: atoiOr("SIMULATE_HUMAN_SEATS", 0),
	}
}

func truthy(s string) bool {
	switch s {
	case "1", "true", "TRUE", "yes", "on":
		return true
	}
	return false
}

func atoiOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// buildNarrator constructs the Anthropic + ElevenLabs backed narrator
// if BOTH credentials are present. Missing either one returns nil and
// the server disables the /host/narrate endpoint; the host client
// handles that gracefully by greying out the Speak button.
func buildNarrator(ctx context.Context) *narrator.Narrator {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		slog.Info("narrator disabled: ANTHROPIC_API_KEY not set")
		return nil
	}
	if os.Getenv("ELEVENLABS_API_KEY") == "" {
		slog.Info("narrator disabled: ELEVENLABS_API_KEY not set")
		return nil
	}
	infer, err := inference.DefaultAnthropicTextInference(ctx)
	if err != nil {
		slog.Warn("narrator init (anthropic) failed", "err", err)
		return nil
	}
	ttsConn, err := tts.DefaultElevenLabsConnector(ctx)
	if err != nil {
		slog.Warn("narrator init (elevenlabs) failed", "err", err)
		return nil
	}
	cfg := narrator.Config{
		Model:    envOr("NARRATOR_MODEL", "claude-sonnet-4-6"),
		VoiceID:  os.Getenv("ELEVENLABS_VOICE_ID"),
		TTSModel: envOr("ELEVENLABS_MODEL_ID", "eleven_turbo_v2_5"),
	}
	slog.Info("narrator ready", "model", cfg.Model, "voice", cfg.VoiceID)
	return narrator.New(infer, ttsConn, cfg)
}

// buildRulesBot constructs the in-game rules helper if ANTHROPIC_API_KEY
// is set. The helper answers natural-language rules questions from
// player phones; no TTS required. Returns nil to disable the feature
// (the mobile help button hides itself).
func buildRulesBot(ctx context.Context) *rulesbot.Bot {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		slog.Info("rulesbot disabled: ANTHROPIC_API_KEY not set")
		return nil
	}
	infer, err := inference.DefaultAnthropicTextInference(ctx)
	if err != nil {
		slog.Warn("rulesbot init (anthropic) failed", "err", err)
		return nil
	}
	cfg := rulesbot.Config{Model: envOr("RULESBOT_MODEL", "claude-sonnet-4-6")}
	slog.Info("rulesbot ready", "model", cfg.Model)
	return rulesbot.New(infer, cfg)
}

// parseCSV splits an env value like "http://a.com, http://b.com" into a
// trimmed slice. Empty input returns nil so AllowedOrigins stays empty
// and the CORS middleware isn't registered at all.
func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func msEnv(key string) time.Duration {
	n := atoiOr(key, 0)
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}
