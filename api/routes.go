package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// registerRoutes wires every endpoint the service exposes.
//
// Routing convention:
//
//	/api/v1/games                          POST  create a new lobby (host)
//	/api/v1/games/join                     POST  join a lobby by code
//	/api/v1/games/:gameId                  GET   fetch current state
//
//	/api/v1/games/:gameId/host/start          POST  host starts the match
//	/api/v1/games/:gameId/host/force-progress POST  host force-advances phase
//	/api/v1/games/:gameId/host/timer-tick     POST  scheduler timer expiry
//
//	/api/v1/games/:gameId/player/nominate     POST  president picks chancellor
//	/api/v1/games/:gameId/player/vote         POST  cast a ja/nein vote
//	/api/v1/games/:gameId/player/discard      POST  president discards a policy
//	/api/v1/games/:gameId/player/enact        POST  chancellor enacts a policy
//	/api/v1/games/:gameId/player/veto         POST  chancellor proposes a veto
//	/api/v1/games/:gameId/player/veto/resolve POST  president accepts/rejects veto
//	/api/v1/games/:gameId/player/action       POST  president uses executive power
//
//	/api/v1/games/:gameId/stream/board        GET   SSE for the host display
//	/api/v1/games/:gameId/stream/player       GET   SSE for a player device
//
// Health: /healthz returns 200 OK with no auth required.
func (s *Server) registerRoutes() {
	s.router.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	v1 := s.router.Group("/api/v1")
	v1.Use(s.requireAuth())

	// Lobby & game lookup
	v1.POST("/games", s.handleCreateGame)
	v1.POST("/games/join", s.handleJoinGame)
	v1.GET("/games/:gameId", s.handleGetGame)

	// Host actions (board device)
	host := v1.Group("/games/:gameId/host")
	host.POST("/start", s.handleStartGame)
	host.POST("/force-progress", s.handleForceProgress)
	host.POST("/timer-tick", s.handleTimerTick)
	host.POST("/narrate", s.handleNarrate)
	host.PUT("/rules", s.handleUpdateRules)

	// Narrator audio cache — broadcast scope, any authenticated
	// device may fetch a cue's MP3 by id.
	v1.GET("/narrator/audio/:cueId", s.handleNarratorAudio)

	// Player actions (player devices)
	player := v1.Group("/games/:gameId/player")
	player.POST("/nominate", s.handleNominateChancellor)
	player.POST("/vote", s.handleCastVote)
	player.POST("/discard", s.handlePresidentDiscard)
	player.POST("/enact", s.handleChancellorEnact)
	player.POST("/veto", s.handleProposeVeto)
	player.POST("/veto/resolve", s.handleResolveVeto)
	player.POST("/action", s.handleExecuteAction)
	player.POST("/chat", s.handleChatSend)
	player.POST("/dm", s.handleDMSend)
	player.GET("/dms", s.handleDMHistory)
	player.POST("/ask", s.handleRulesAsk)
	player.GET("/faq", s.handleRulesFAQ)

	// Streams (SSE). Auth middleware applies here too.
	stream := v1.Group("/games/:gameId/stream")
	stream.GET("/board", s.handleBoardStream)
	stream.GET("/player", s.handlePlayerStream)
}
