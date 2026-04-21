// Package api exposes the Replicant backend over HTTP using Gin.
//
// The API is split into two device families:
//
//   - Board device: the host's "game board" display. It joins the
//     game via /host/* endpoints (create, start, force-progress) and
//     subscribes to the broadcast-only stream at /stream/board.
//
//   - Player device: each player's phone. Authenticates as a player,
//     performs role-specific actions (nominate, vote, discard, enact,
//     veto, executive action), and subscribes to /stream/player to
//     receive both public updates and secret whispers (role, drawn
//     policies, investigation results).
//
// Streaming is Server-Sent Events backed by handlers/game.Hub so the
// transport can be swapped for WebSockets or Redis pub-sub without
// touching the state machine.
package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/handlers/game"
	"github.com/cannonball10/foundation/handlers/narrator"
	"github.com/cannonball10/foundation/handlers/rulesbot"
	"github.com/gin-gonic/gin"
)

// Server bundles the game engine, hub, and HTTP router.
type Server struct {
	engine        *game.GameHandler
	hub           game.Hub
	router        *gin.Engine
	auth          Authenticator
	users         *userResolver
	db            database.DatabaseConnector
	simulate      bool
	simConfig     game.SimulationConfig
	narrator      *narrator.Narrator // optional; nil when LLM/TTS creds missing
	narratorCache *narratorCache
	rulesbot      *rulesbot.Bot // optional; nil when ANTHROPIC_API_KEY missing
}

// Options configures a Server.
type Options struct {
	// Database is the backing store used by the engine.
	Database database.DatabaseConnector

	// Auth is the authenticator used to resolve the caller's user ID
	// from an incoming request. NopAuth is a reasonable default for
	// local development.
	Auth Authenticator

	// Hub, if nil, defaults to a local in-memory hub.
	Hub game.Hub

	// EngineOptions are forwarded to NewGameHandler (clock, rng, config).
	EngineOptions []game.Option

	// GinMode sets gin's mode ("debug", "release", "test"). Defaults to release.
	GinMode string

	// SimulateOnCreate, when true, fires a bot-driven playthrough in a
	// background goroutine every time CreateGame succeeds. Dev-only:
	// pairs with game.SimulateGame to let the board device watch a full
	// game without needing human players. Enable via SIMULATE_ON_CREATE=1.
	SimulateOnCreate bool

	// SimulateConfig tunes the playthrough when SimulateOnCreate is true.
	// Zero-value fields fall back to game.SimulationConfig defaults.
	SimulateConfig game.SimulationConfig

	// AllowedOrigins, when non-empty, enables a CORS middleware that
	// echoes back origins matching this list. An entry of "*" opens
	// the server to any origin (dev only). Leave empty to rely on
	// same-origin deployment or a front proxy / Next.js rewrite to
	// handle CORS externally.
	AllowedOrigins []string

	// Narrator, when set, exposes POST /host/narrate and
	// GET /narrator/audio/:cueId. Leave nil to disable the narrator
	// (host screen will still render a greyed-out SPEAK button).
	Narrator *narrator.Narrator

	// RulesBot, when set, exposes POST /player/ask — a natural-
	// language rules-and-roles helper on mobile. Leave nil to
	// disable (the mobile UI hides the help button).
	RulesBot *rulesbot.Bot
}

// NewServer constructs and wires a ready-to-serve Server.
func NewServer(opts Options) *Server {
	if opts.Hub == nil {
		opts.Hub = game.NewMemoryHub()
	}
	if opts.Auth == nil {
		opts.Auth = NopAuth{}
	}
	if opts.GinMode == "" {
		opts.GinMode = gin.ReleaseMode
	}
	gin.SetMode(opts.GinMode)

	// Engine is wired to emit through the hub. If a narrator is
	// configured, thread it into the engine so Cable Phase leaks can
	// call RankCables + Speak without an extra layer.
	engineOpts := append([]game.Option{
		game.WithEmitter(game.NewHubEmitter(opts.Hub)),
	}, opts.EngineOptions...)
	if opts.Narrator != nil {
		engineOpts = append(engineOpts, game.WithCableNarrator(opts.Narrator))
	}
	engine := game.NewGameHandler(opts.Database, engineOpts...)

	s := &Server{
		engine:        engine,
		hub:           opts.Hub,
		auth:          opts.Auth,
		users:         newUserResolver(opts.Database),
		db:            opts.Database,
		router:        gin.New(),
		simulate:      opts.SimulateOnCreate,
		simConfig:     opts.SimulateConfig,
		narrator:      opts.Narrator,
		narratorCache: newNarratorCache(5 * time.Minute),
		rulesbot:      opts.RulesBot,
	}
	s.router.Use(gin.Recovery())
	if len(opts.AllowedOrigins) > 0 {
		s.router.Use(corsMiddleware(opts.AllowedOrigins))
	}
	s.registerRoutes()
	return s
}

// corsMiddleware is a minimal CORS layer. It echoes back the request's
// Origin when it's in the allow-list (or when "*" is allow-listed) and
// handles OPTIONS preflights. Tokens may travel on the query string for
// SSE, so we explicitly allow that header family plus Authorization.
//
// Wildcard entries are supported for "any subdomain of X" patterns —
// "https://*.ngrok-free.dev" matches every ephemeral ngrok tunnel
// without needing to update .env every time ngrok rolls the URL.
// The leftmost label is the only one that's a wildcard; the scheme
// and the rest of the host must match exactly.
func corsMiddleware(allowed []string) gin.HandlerFunc {
	allowAll := false
	exact := make(map[string]bool, len(allowed))
	var wildcards []string
	for _, o := range allowed {
		if o == "*" {
			allowAll = true
			continue
		}
		norm := strings.ToLower(strings.TrimRight(o, "/"))
		if strings.Contains(norm, "://*.") {
			// Store the suffix that every match must end with,
			// including the scheme up to and including "://".
			i := strings.Index(norm, "://*.")
			scheme := norm[:i+len("://")]
			suffix := norm[i+len("://*."):]
			wildcards = append(wildcards, scheme+"|."+suffix)
			continue
		}
		exact[norm] = true
	}
	return func(c *gin.Context) {
		origin := strings.ToLower(c.GetHeader("Origin"))
		if origin != "" && (allowAll || exact[origin] || matchesWildcard(origin, wildcards)) {
			c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header(
				"Access-Control-Allow-Headers",
				"Authorization, Content-Type, X-Auth-Token",
			)
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// matchesWildcard tests whether origin matches any pattern of the
// form "scheme://|.suffix" (produced by corsMiddleware). Encoded
// this way so a single pass can check both the scheme match and the
// suffix match — we intentionally do NOT let "http://*.example.com"
// allow "https://...example.com" since the schemes differ.
func matchesWildcard(origin string, patterns []string) bool {
	for _, p := range patterns {
		pipe := strings.Index(p, "|")
		if pipe <= 0 {
			continue
		}
		scheme, suffix := p[:pipe], p[pipe+1:]
		if strings.HasPrefix(origin, scheme) && strings.HasSuffix(origin, suffix) {
			return true
		}
	}
	return false
}

// Handler exposes the underlying http.Handler so the caller can mount
// it in their own listener (or compose with other middleware).
func (s *Server) Handler() http.Handler { return s.router }

// Run starts the HTTP server on addr (":8080" by default). It blocks
// until the server stops.
func (s *Server) Run(addr string) error {
	if addr == "" {
		addr = ":8080"
	}
	return s.router.Run(addr)
}

// Shutdown is a placeholder for future graceful-shutdown logic. It
// currently just allows callers to close their dependencies; the Gin
// router itself stops when the listener is closed by the caller.
func (s *Server) Shutdown(_ context.Context) error { return nil }
