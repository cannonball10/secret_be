// Package api exposes the Secret Hitler backend over HTTP using Gin.
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

	"github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/handlers/game"
	"github.com/gin-gonic/gin"
)

// Server bundles the game engine, hub, and HTTP router.
type Server struct {
	engine *game.GameHandler
	hub    game.Hub
	router *gin.Engine
	auth   Authenticator
	db     database.DatabaseConnector
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

	// Engine is wired to emit through the hub.
	engineOpts := append([]game.Option{
		game.WithEmitter(game.NewHubEmitter(opts.Hub)),
	}, opts.EngineOptions...)
	engine := game.NewGameHandler(opts.Database, engineOpts...)

	s := &Server{
		engine: engine,
		hub:    opts.Hub,
		auth:   opts.Auth,
		db:     opts.Database,
		router: gin.New(),
	}
	s.router.Use(gin.Recovery())
	s.registerRoutes()
	return s
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
