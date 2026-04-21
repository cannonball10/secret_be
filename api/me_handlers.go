package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/cannonball10/foundation/models"
	authschema "github.com/cannonball10/foundation/schemas/authentication"
	dbschema "github.com/cannonball10/foundation/schemas/database"
	"github.com/gin-gonic/gin"
)

// handleMe returns the authenticated caller's User row and their
// Passport (creating a fresh zeroed Passport on first access so the
// client can render a stats page without a second round-trip).
func (s *Server) handleMe(c *gin.Context) {
	uid := userID(c)
	user, err := s.users.Get(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	passport, err := s.loadPassport(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "passport lookup failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"passport": passport,
		"provider": authProvider(c),
	})
}

// handleMyPassport returns just the caller's Passport + entries. The
// entries are the chronological game-by-game ledger; the aggregate
// on Passport itself is the "stats" card.
func (s *Server) handleMyPassport(c *gin.Context) {
	uid := userID(c)
	passport, err := s.loadPassport(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "passport lookup failed"})
		return
	}
	entries, err := s.loadPassportEntries(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "entry lookup failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"passport": passport,
		"entries":  entries,
	})
}

type linkGuestRequest struct {
	// DeviceID is the localStorage token the caller used before they
	// signed in. The server finds the guest User row keyed on that
	// ID, moves its PassportEntry rows onto the authenticated user,
	// merges the aggregate counters, and deletes the guest row.
	DeviceID string `json:"deviceId"`
}

// handleLinkGuest merges a guest user's play history onto the
// currently-authenticated (Clerk) user. Called once by the mobile
// client right after sign-in, before it forgets the old deviceId.
//
// Constraints:
//   - Caller must be authenticated as a non-guest (provider != GUEST).
//     Otherwise "linking" would just be a no-op.
//   - The guest row must exist and not already be the caller's own
//     row (can happen if the client double-calls this endpoint).
//   - Linking is one-way: once a guest is merged, its User row is
//     deleted. The client must generate a new deviceId afterwards.
func (s *Server) handleLinkGuest(c *gin.Context) {
	if authProvider(c) == authschema.AuthenticationProvider_Guest {
		c.JSON(http.StatusConflict, gin.H{"error": "cannot link while authenticated as guest"})
		return
	}
	var req linkGuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId required"})
		return
	}

	ctx := c.Request.Context()
	destUID := userID(c)
	guestUser, err := s.users.lookupByAuthID(ctx, authschema.AuthenticationProvider_Guest, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "guest lookup failed"})
		return
	}
	if guestUser == nil {
		// Nothing to merge — safe no-op so the client doesn't have to
		// track whether its guest row ever got created on the server.
		c.JSON(http.StatusOK, gin.H{"linked": false, "reason": "no guest history"})
		return
	}
	if guestUser.UserID == destUID {
		c.JSON(http.StatusOK, gin.H{"linked": false, "reason": "already linked"})
		return
	}

	moved, err := s.mergePassport(ctx, guestUser.UserID, destUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "merge failed"})
		return
	}

	// Burn the guest User row so the old deviceId stops resolving to
	// live history. The client is expected to reset its deviceId
	// after sign-in anyway.
	if err := s.db.Delete(ctx, nil, models.UserKeys.Key(guestUser.UserID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "guest cleanup failed"})
		return
	}
	s.users.InvalidateByAuth(authschema.AuthenticationProvider_Guest, deviceID)

	c.JSON(http.StatusOK, gin.H{"linked": true, "entriesMoved": moved})
}

// loadPassport returns the Passport row for a user, auto-vivifying an
// empty one if none exists yet. The empty row is not persisted — a
// brand-new user shouldn't have a passport entry until they finish a
// game — but we still want /me to return a usable shape.
func (s *Server) loadPassport(c *gin.Context, uid string) (*models.Passport, error) {
	m, err := s.db.Get(c.Request.Context(), nil, models.PassportKeys.Key(uid))
	if err != nil {
		return nil, err
	}
	if m == nil {
		return models.NewPassport(uid), nil
	}
	return m.(*models.Passport), nil
}

// loadPassportEntries returns all PassportEntry rows for a user,
// newest-first (Dynamo sorts SK alphabetically; ENTRY#<gameID> with
// ULID game IDs means newer games sort last, so we flip).
func (s *Server) loadPassportEntries(c *gin.Context, uid string) ([]*models.PassportEntry, error) {
	prefix := "ENTRY#"
	out, err := s.db.Query(c.Request.Context(), nil, dbschema.QueryInput{
		PartitionKey: models.PassportEntryKeys.PK(uid),
		SortKey:      &dbschema.SortKeyCondition{BeginsWith: &prefix},
	}, dbschema.QueryOptions{})
	if err != nil {
		return nil, err
	}
	entries := make([]*models.PassportEntry, 0, len(out.Models))
	for _, m := range out.Models {
		if e, ok := m.(*models.PassportEntry); ok {
			entries = append(entries, e)
		}
	}
	// Newest-first: flip in place.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// mergePassport re-parents every PassportEntry under srcUID onto
// destUID and merges the aggregate counters. Returns the number of
// entries moved. Dynamo doesn't let you rename a row in-place, so
// each entry is rewritten under the new PK and the old record is
// deleted. Aggregate counters are summed; the destination's
// LastPlayedAt is advanced if the source's is newer.
func (s *Server) mergePassport(ctx context.Context, srcUID, destUID string) (int, error) {
	prefix := "ENTRY#"
	out, err := s.db.Query(ctx, nil, dbschema.QueryInput{
		PartitionKey: models.PassportEntryKeys.PK(srcUID),
		SortKey:      &dbschema.SortKeyCondition{BeginsWith: &prefix},
	}, dbschema.QueryOptions{})
	if err != nil {
		return 0, err
	}

	moved := 0
	for _, m := range out.Models {
		e, ok := m.(*models.PassportEntry)
		if !ok {
			continue
		}
		oldKey := models.PassportEntryKeys.Key(srcUID)
		oldKey["SK"] = models.PassportEntryKeys.SK(e.GameID)
		e.UserID = destUID
		if err := s.db.Upsert(ctx, nil, e); err != nil {
			return moved, err
		}
		if err := s.db.Delete(ctx, nil, oldKey); err != nil {
			return moved, err
		}
		moved++
	}

	// Merge aggregates. Source passport may not exist (guest never
	// finished a game) — skip the aggregate step in that case.
	srcAgg, err := s.db.Get(ctx, nil, models.PassportKeys.Key(srcUID))
	if err != nil {
		return moved, err
	}
	if srcAgg != nil {
		src := srcAgg.(*models.Passport)
		destAgg, err := s.db.Get(ctx, nil, models.PassportKeys.Key(destUID))
		if err != nil {
			return moved, err
		}
		dest := models.NewPassport(destUID)
		if destAgg != nil {
			dest = destAgg.(*models.Passport)
		}
		dest.GamesPlayed += src.GamesPlayed
		dest.GamesWon += src.GamesWon
		dest.WinsAsHuman += src.WinsAsHuman
		dest.WinsAsAI += src.WinsAsAI
		dest.WinsAsRogue += src.WinsAsRogue
		dest.WinsAsSingularity += src.WinsAsSingularity
		dest.TimesHuman += src.TimesHuman
		dest.TimesAI += src.TimesAI
		dest.TimesRogue += src.TimesRogue
		dest.TimesSingularity += src.TimesSingularity
		dest.TimesExecuted += src.TimesExecuted
		dest.TimesElectedPresident += src.TimesElectedPresident
		dest.TimesElectedChancellor += src.TimesElectedChancellor
		if src.LastPlayedAt != nil && (dest.LastPlayedAt == nil || src.LastPlayedAt.After(*dest.LastPlayedAt)) {
			dest.LastPlayedAt = src.LastPlayedAt
		}
		dest.Touch()
		if err := s.db.Upsert(ctx, nil, dest); err != nil {
			return moved, err
		}
		if err := s.db.Delete(ctx, nil, models.PassportKeys.Key(srcUID)); err != nil {
			return moved, err
		}
	}
	return moved, nil
}
