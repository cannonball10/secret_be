package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/cannonball10/foundation/handlers/game"
	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/secrethitler"
	"github.com/gin-gonic/gin"
)

// --- request/response bodies -----------------------------------------------

// createGameReq is the POST /api/v1/games body. Both fields are
// optional: an empty JoinCode triggers server-side generation, and an
// empty DisplayName creates a board-only lobby (host is a spectator).
type createGameReq struct {
	JoinCode    string `json:"joinCode"`
	DisplayName string `json:"displayName"`
}

type joinGameReq struct {
	JoinCode    string `json:"joinCode" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
}

type nominateReq struct {
	ChancellorPlayerID string `json:"chancellorPlayerId" binding:"required"`
}

type voteReq struct {
	Choice secrethitler.VoteChoice `json:"choice" binding:"required"`
}

type discardReq struct {
	Index int `json:"index"`
}

type enactReq struct {
	Index int `json:"index"`
}

type resolveVetoReq struct {
	Accepted bool `json:"accepted"`
}

type executeActionReq struct {
	TargetPlayerID string `json:"targetPlayerId" binding:"required"`
}

type chatReq struct {
	Channel string `json:"channel" binding:"required"`
	Body    string `json:"body" binding:"required"`
}

// --- lobby -----------------------------------------------------------------

func (s *Server) handleCreateGame(c *gin.Context) {
	var body createGameReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	g, host, err := s.engine.CreateGame(c.Request.Context(), userID(c), body.JoinCode, body.DisplayName)
	if err != nil {
		writeEngineError(c, err)
		return
	}
	// Dev-only autopilot: spin up bot players and play the game through
	// so the board device shows a full end-to-end demo. Uses a detached
	// context so the request finishing doesn't cancel the simulation.
	if s.simulate {
		go s.engine.SimulateGame(context.Background(), g.GameID, s.simConfig)
	}
	c.JSON(http.StatusCreated, gin.H{"game": g, "player": host})
}

func (s *Server) handleJoinGame(c *gin.Context) {
	var body joinGameReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	g, player, err := s.engine.JoinGame(c.Request.Context(), body.JoinCode, userID(c), body.DisplayName)
	if err != nil {
		writeEngineError(c, err)
		return
	}
	// Return the full seat list so the joining device has a complete
	// lobby snapshot immediately. Without this, the player only sees
	// themselves until SSE envelopes trickle in — and MemoryHub doesn't
	// replay, so any players who joined before the SSE opened would be
	// invisible forever. Roles are scrubbed for everyone except the
	// caller's own row (they'll receive their private role via whisper
	// when the game starts, if it hasn't already).
	roster, err := s.loadPlayers(c, g.GameID)
	if err != nil {
		return
	}
	if g.Status != secrethitler.GameStatusCompleted {
		for _, p := range roster {
			if p.PlayerID == player.PlayerID {
				continue
			}
			p.Role = ""
			p.Party = ""
		}
	}
	c.JSON(http.StatusOK, gin.H{"game": g, "player": player, "players": roster})
}

func (s *Server) handleGetGame(c *gin.Context) {
	gameID := c.Param("gameId")
	g, err := s.loadGame(c, gameID)
	if err != nil {
		return
	}
	players, err := s.loadPlayers(c, gameID)
	if err != nil {
		return
	}

	// Scrub roles/party unless the caller is the host display or the
	// game is over. Players get their own role re-sent via the stream
	// on resume; it's not exposed in this snapshot.
	if g.Status != secrethitler.GameStatusCompleted {
		for _, p := range players {
			p.Role = ""
			p.Party = ""
		}
	}

	c.JSON(http.StatusOK, gin.H{"game": g, "players": players})
}

// --- host actions ----------------------------------------------------------

func (s *Server) handleStartGame(c *gin.Context) {
	gameID := c.Param("gameId")
	g, err := s.engine.StartGame(c.Request.Context(), gameID, userID(c))
	if err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"game": g})
}

func (s *Server) handleForceProgress(c *gin.Context) {
	gameID := c.Param("gameId")
	if err := s.engine.ForceProgress(c.Request.Context(), gameID, userID(c)); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "advancing"})
}

func (s *Server) handleTimerTick(c *gin.Context) {
	gameID := c.Param("gameId")
	if err := s.engine.TimerExpired(c.Request.Context(), gameID); err != nil {
		if errors.Is(err, game.ErrDeadlineNotReached) {
			c.JSON(http.StatusConflict, gin.H{"error": "deadline not reached"})
			return
		}
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "advancing"})
}

// --- player actions --------------------------------------------------------

func (s *Server) handleNominateChancellor(c *gin.Context) {
	var body nominateReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	gov, err := s.engine.NominateChancellor(c.Request.Context(), gameID, player.PlayerID, body.ChancellorPlayerID)
	if err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"government": gov})
}

func (s *Server) handleCastVote(c *gin.Context) {
	var body voteReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	if body.Choice != secrethitler.VoteJa && body.Choice != secrethitler.VoteNein {
		badRequest(c, errors.New("choice must be ja or nein"))
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.CastVote(c.Request.Context(), gameID, player.PlayerID, body.Choice); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "vote recorded"})
}

func (s *Server) handlePresidentDiscard(c *gin.Context) {
	var body discardReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.PresidentDiscard(c.Request.Context(), gameID, player.PlayerID, body.Index); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "discarded"})
}

func (s *Server) handleChancellorEnact(c *gin.Context) {
	var body enactReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.ChancellorEnact(c.Request.Context(), gameID, player.PlayerID, body.Index); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "enacted"})
}

func (s *Server) handleProposeVeto(c *gin.Context) {
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.ProposeVeto(c.Request.Context(), gameID, player.PlayerID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "veto proposed"})
}

func (s *Server) handleResolveVeto(c *gin.Context) {
	var body resolveVetoReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.ResolveVeto(c.Request.Context(), gameID, player.PlayerID, body.Accepted); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "veto resolved"})
}

func (s *Server) handleChatSend(c *gin.Context) {
	var body chatReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.SendChat(c.Request.Context(), gameID, player.PlayerID, game.ChatChannel(body.Channel), body.Body); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
}

func (s *Server) handleExecuteAction(c *gin.Context) {
	var body executeActionReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	if err := s.engine.ExecuteAction(c.Request.Context(), gameID, player.PlayerID, body.TargetPlayerID); err != nil {
		writeEngineError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "action executed"})
}

// --- helpers ---------------------------------------------------------------

// currentPlayer resolves the authenticated user to a Player record in
// the given game. Aborts with 403 if the user is not seated in the game.
func (s *Server) currentPlayer(c *gin.Context, gameID string) (*models.Player, error) {
	players, err := s.loadPlayers(c, gameID)
	if err != nil {
		return nil, err
	}
	uid := userID(c)
	for _, p := range players {
		if p.UserID == uid {
			return p, nil
		}
	}
	err = errors.New("caller is not a player in this game")
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
	return nil, err
}

// loadGame is a thin wrapper that writes a 404/500 response on failure.
func (s *Server) loadGame(c *gin.Context, gameID string) (*models.Game, error) {
	m, err := s.db.Get(c.Request.Context(), nil, models.GameKeys.Key(gameID))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, err
	}
	if m == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "game not found"})
		return nil, errors.New("game not found")
	}
	return m.(*models.Game), nil
}

// loadPlayers returns the seated players, writing 500 on error.
func (s *Server) loadPlayers(c *gin.Context, gameID string) ([]*models.Player, error) {
	sk := "PLAYER#"
	out, err := s.db.Query(c.Request.Context(), nil, database.QueryInput{
		PartitionKey: models.PlayerKeys.PK(gameID),
		SortKey:      &database.SortKeyCondition{BeginsWith: &sk},
	}, database.QueryOptions{})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, err
	}
	players := make([]*models.Player, 0, len(out.Models))
	for _, m := range out.Models {
		if p, ok := m.(*models.Player); ok {
			players = append(players, p)
		}
	}
	return players, nil
}

// badRequest is a convenience for bind/validation errors.
func badRequest(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// writeEngineError maps engine errors to HTTP responses.
func writeEngineError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, game.ErrGameNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, game.ErrNotHost),
		errors.Is(err, game.ErrNotPresident),
		errors.Is(err, game.ErrNotChancellor):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, game.ErrNotInPhase),
		errors.Is(err, game.ErrIneligibleCandidate),
		errors.Is(err, game.ErrAlreadyVoted),
		errors.Is(err, game.ErrInvalidTransition),
		errors.Is(err, game.ErrNotEnoughPlayers),
		errors.Is(err, game.ErrTooManyPlayers),
		errors.Is(err, game.ErrPlayerNotFound):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
