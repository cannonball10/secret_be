package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cannonball10/foundation/handlers/game"
	"github.com/gin-gonic/gin"
)

// handleBoardStream opens an SSE stream scoped to the host's game-board
// display. It only receives broadcast envelopes; whispers (secret
// information meant for a specific player) are never delivered here.
// Only the game host may open a board stream.
func (s *Server) handleBoardStream(c *gin.Context) {
	gameID := c.Param("gameId")
	g, err := s.loadGame(c, gameID)
	if err != nil {
		return
	}
	if g.HostUserID != userID(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "only the host can open a board stream"})
		return
	}
	sub := s.hub.Subscribe(gameID, game.DeviceRoleBoard, "")
	defer s.hub.Unsubscribe(gameID, sub.ID)
	streamEnvelopes(c, sub)
}

// handlePlayerStream opens an SSE stream for a player's phone. The
// stream receives every broadcast envelope for the game plus any
// whisper addressed to the caller's player record (private role,
// drawn policies, investigation results, policy peek).
func (s *Server) handlePlayerStream(c *gin.Context) {
	gameID := c.Param("gameId")
	player, err := s.currentPlayer(c, gameID)
	if err != nil {
		return
	}
	sub := s.hub.Subscribe(gameID, game.DeviceRolePlayer, player.PlayerID)
	defer s.hub.Unsubscribe(gameID, sub.ID)
	streamEnvelopes(c, sub)
}

// streamEnvelopes writes Server-Sent Events to the response until the
// client disconnects or the subscription channel closes. Each event
// looks like:
//
//	id: <subId>-<seq>
//	event: <eventType>
//	data: <json envelope>
//
// Clients should use the EventSource API in browsers or any SSE-
// compatible library on native mobile.
func streamEnvelopes(c *gin.Context, sub *game.Subscriber) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering
	c.Writer.WriteHeader(http.StatusOK)

	// Kick off with a "hello" comment so proxies flush headers and the
	// client knows the stream is live.
	if _, err := io.WriteString(c.Writer, ": connected\n\n"); err != nil {
		return
	}
	c.Writer.Flush()

	ctx := c.Request.Context()
	seq := 0
	c.Stream(func(w io.Writer) bool {
		select {
		case env, ok := <-sub.Send:
			if !ok {
				return false
			}
			seq++
			if err := writeSSE(w, sub.ID, seq, env); err != nil {
				return false
			}
			return true
		case <-ctx.Done():
			return false
		}
	})
}

// writeSSE serializes an envelope in SSE frame format.
func writeSSE(w io.Writer, subID string, seq int, env game.Envelope) error {
	payload, err := json.Marshal(env)
	if err != nil {
		return err
	}
	eventType := "message"
	if env.Event != nil {
		eventType = string(env.Event.Type)
	}
	_, err = fmt.Fprintf(w, "id: %s-%d\nevent: %s\ndata: %s\n\n", subID, seq, eventType, payload)
	return err
}
