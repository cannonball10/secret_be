package api

import (
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/cannonball10/foundation/handlers/game"
	"github.com/cannonball10/foundation/handlers/narrator"
	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/replicant"
	"github.com/gin-gonic/gin"
)

// narratorCache holds synthesised audio briefly so the host can fetch
// it via GET /narrator/audio/:cueId and so mobile spectators can
// optionally play the same clip. Entries are evicted after the TTL;
// a single running game produces at most a few cues per minute so an
// in-memory map is plenty.
type narratorCache struct {
	mu  sync.Mutex
	ix  map[string]narratorCacheEntry
	ttl time.Duration
}

type narratorCacheEntry struct {
	audio    []byte
	mime     string
	addedAt  time.Time
}

func newNarratorCache(ttl time.Duration) *narratorCache {
	return &narratorCache{ix: make(map[string]narratorCacheEntry), ttl: ttl}
}

func (c *narratorCache) put(id, mime string, audio []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Opportunistic GC: every put walks + expires stale entries. With
	// at most a few dozen entries per session this is a tiny cost.
	now := time.Now()
	for k, v := range c.ix {
		if now.Sub(v.addedAt) > c.ttl {
			delete(c.ix, k)
		}
	}
	c.ix[id] = narratorCacheEntry{audio: audio, mime: mime, addedAt: now}
}

func (c *narratorCache) get(id string) (narratorCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.ix[id]
	if !ok {
		return narratorCacheEntry{}, false
	}
	if time.Since(e.addedAt) > c.ttl {
		delete(c.ix, id)
		return narratorCacheEntry{}, false
	}
	return e, true
}

// narrateReq is the POST /host/narrate body. Host supplies the cue
// kind + optional inline text (for CueCustom) + optional override
// variables (e.g. name of executed player when the cue is "execution").
type narrateReq struct {
	Cue  string            `json:"cue" binding:"required"`
	Vars map[string]string `json:"vars,omitempty"`
	Text string            `json:"text,omitempty"`
}

func (s *Server) handleNarrate(c *gin.Context) {
	if s.narrator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "narrator not configured: set ANTHROPIC_API_KEY + ELEVENLABS_API_KEY",
		})
		return
	}

	gameID := c.Param("gameId")
	g, err := s.loadGame(c, gameID)
	if err != nil {
		return
	}
	if g.HostUserID != userID(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "host only"})
		return
	}
	// Honour the host's opt-out. The host UI reads the same flag and
	// greys out SPEAK, but a stale button click or an auto-cue still
	// hits here — reject cleanly so the voice stays silent.
	if g.Rules.DisableNarrator {
		c.JSON(http.StatusConflict, gin.H{"error": "narrator disabled by rules"})
		return
	}

	var body narrateReq
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, err)
		return
	}

	cue := narrator.Cue{
		Kind: narrator.CueKind(body.Cue),
		Vars: map[string]string{},
	}
	for k, v := range body.Vars {
		cue.Vars[k] = v
	}
	// Server-side enrichment: fill common variables the client would
	// otherwise have to look up. For CueOpening we add playerCount.
	if cue.Kind == narrator.CueOpening {
		if _, ok := cue.Vars["playerCount"]; !ok {
			players, err := s.loadPlayers(c, gameID)
			if err != nil {
				return
			}
			cue.Vars["playerCount"] = formatInt(len(players))
		}
	}
	if body.Text != "" {
		cue.Vars["text"] = body.Text
	}

	result, err := s.narrator.Speak(c.Request.Context(), cue)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	mime := "audio/mpeg"
	s.narratorCache.put(result.CueID, mime, result.Audio)

	// Emit a broadcast envelope. Audio is sent as a data URL in the
	// payload so mobile clients can play too without a separate GET.
	// Host clients also have the dedicated audio endpoint as a
	// fallback for cache validation.
	audioURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(result.Audio)
	ev := models.NewGameEvent(gameID, replicant.EventNarratorSpeak, "")
	payload := game.NarratorSpeakPayload{
		CueID:    result.CueID,
		Cue:      body.Cue,
		Script:   result.Script,
		AudioURL: audioURL,
		AudioID:  result.CueID,
		Format:   string(result.Format),
		TookMS:   result.TookMS,
	}
	s.hub.Publish(c.Request.Context(), game.Envelope{
		GameID:   gameID,
		Event:    ev,
		Audience: game.Audience{Scope: game.AudienceBroadcast},
		Payload:  payload,
	})

	// Also return the payload in the HTTP response so the caller has
	// a sync view of what was just sent.
	c.JSON(http.StatusOK, payload)
}

// handleNarratorAudio serves cached MP3 bytes for a given cue id.
// Open route (behind auth middleware) — any authenticated player in
// the game can pull the audio, which matches the broadcast audience
// of the narrator_speak envelope.
func (s *Server) handleNarratorAudio(c *gin.Context) {
	id := c.Param("cueId")
	if s.narratorCache == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	e, ok := s.narratorCache.get(id)
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "audio expired or unknown"})
		return
	}
	c.Writer.Header().Set("Cache-Control", "private, max-age=300")
	c.Data(http.StatusOK, e.mime, e.audio)
}

// formatInt avoids pulling strconv in for one call.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append([]byte{'0' + byte(n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

