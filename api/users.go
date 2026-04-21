package api

import (
	"context"
	"fmt"
	"sync"

	dbconn "github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/models"
	authschema "github.com/cannonball10/foundation/schemas/authentication"
	dbschema "github.com/cannonball10/foundation/schemas/database"
)

// userResolver maps an authenticated external identity
// (provider + externalID) to our own internal User row. If a row
// doesn't exist yet, it creates one. Cached in-process to avoid a
// DynamoDB round-trip on every request — in prod a token holds for
// ~60s, so even a tiny cache eliminates the common case.
//
// The resolver intentionally does NOT depend on the full
// handlers/user package — that pulls in asynq + connectors, which is
// overkill for a two-query lookup.
type userResolver struct {
	db dbconn.DatabaseConnector

	mu    sync.Mutex
	cache map[string]string // "provider|externalID" -> internal userId
}

func newUserResolver(db dbconn.DatabaseConnector) *userResolver {
	return &userResolver{
		db:    db,
		cache: make(map[string]string, 64),
	}
}

// Resolve returns the internal User.UserID for the given identity,
// creating a fresh User row if none exists. Thread-safe; concurrent
// callers for the same external ID may both hit the DB once but will
// converge on the same row via the GSI1 unique-ish constraint (last
// writer wins, and both writers produce equivalent rows).
func (r *userResolver) Resolve(ctx context.Context, provider authschema.AuthenticationProvider, externalID string) (*models.User, error) {
	if externalID == "" || provider == "" {
		return nil, fmt.Errorf("userResolver: empty provider or externalID")
	}
	cacheKey := string(provider) + "|" + externalID

	r.mu.Lock()
	cached, ok := r.cache[cacheKey]
	r.mu.Unlock()
	if ok {
		// Hit: we cached the internal userID. Re-fetch the row because
		// /me wants the full record (email, displayName, role).
		got, err := r.db.Get(ctx, nil, models.UserKeys.Key(cached))
		if err != nil {
			return nil, err
		}
		if got != nil {
			return got.(*models.User), nil
		}
		// Cache was stale — fall through and requery the GSI.
	}

	found, err := r.lookupByAuthID(ctx, provider, externalID)
	if err != nil {
		return nil, err
	}
	if found != nil {
		r.cachePut(cacheKey, found.UserID)
		return found, nil
	}

	// Not found — create. For Clerk users we'll backfill email/name
	// later via a webhook or /me PUT; for now an empty display name
	// is fine because the mobile join flow uses Player.DisplayName,
	// not User.DisplayName.
	u := models.NewUser(nil, provider, externalID, "", "", models.UserRole_User)
	if err := r.db.Upsert(ctx, nil, u); err != nil {
		return nil, err
	}
	r.cachePut(cacheKey, u.UserID)
	return u, nil
}

// Get returns an existing User by internal UserID. Used by /me.
func (r *userResolver) Get(ctx context.Context, userID string) (*models.User, error) {
	got, err := r.db.Get(ctx, nil, models.UserKeys.Key(userID))
	if err != nil {
		return nil, err
	}
	if got == nil {
		return nil, nil
	}
	return got.(*models.User), nil
}

// InvalidateByAuth drops any cached entry for a given identity. Used
// by /me/link so a guest that just linked into a signed-in account
// doesn't keep resolving to the orphaned guest row for another 60s.
func (r *userResolver) InvalidateByAuth(provider authschema.AuthenticationProvider, externalID string) {
	r.mu.Lock()
	delete(r.cache, string(provider)+"|"+externalID)
	r.mu.Unlock()
}

func (r *userResolver) cachePut(key, userID string) {
	r.mu.Lock()
	// Bound the cache crudely — if we somehow blow past 4k entries
	// (mostly guests that each get a distinct deviceId), wipe and
	// let warm-up rebuild it. Prevents unbounded growth without
	// shipping a full LRU.
	if len(r.cache) >= 4096 {
		r.cache = make(map[string]string, 64)
	}
	r.cache[key] = userID
	r.mu.Unlock()
}

func (r *userResolver) lookupByAuthID(ctx context.Context, provider authschema.AuthenticationProvider, externalID string) (*models.User, error) {
	indexName := dbschema.IndexName_GSI1
	sk := models.UserGSI1Keys.SK(externalID)
	out, err := r.db.Query(ctx, nil, dbschema.QueryInput{
		PartitionKey: models.UserGSI1Keys.PK(string(provider)),
		SortKey:      &dbschema.SortKeyCondition{EQ: &sk},
		IndexName:    &indexName,
	}, dbschema.QueryOptions{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(out.Models) == 0 {
		return nil, nil
	}
	return out.Models[0].(*models.User), nil
}
