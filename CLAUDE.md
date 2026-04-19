# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project summary

This repo hosts a Go backend for **Secret Hitler** (the board game) plus a
minimal React + TypeScript web client. The backend reuses the existing
"foundation" connector pattern the repo was originally scaffolded around.

The backend implements a full rules-correct engine for 5-10 player
games, streams state over Server-Sent Events, and exposes a Gin HTTP
API. The web client is a dead-simple React shell that consumes both.

Branch: `claude/secret-hitler-data-models-CiDqk`. All work has been
pushed to origin.

---

## Build & Development Commands

```bash
# Build ALL packages. NOTE: connectors/livekit currently has unresolved
# deps; scoped builds below are what you should use day-to-day.
go build ./...

# Build only the packages Secret Hitler actually uses (recommended):
go build ./cmd/api ./api/... ./handlers/game/... ./models/... ./schemas/secrethitler/...

# Test the same:
go test  ./api/... ./handlers/game/... ./models/... ./schemas/secrethitler/...

# Run the backend (see cmd/api/main.go):
cp .env.example .env
docker compose up -d dynamodb
./scripts/create-table.sh
go run ./cmd/api     # :8080

# Run the web client:
cd web
cp .env.example .env.local
npm install
npm run dev          # :5173, proxies /api and /healthz to :8080
npm run typecheck
npm run test         # node --test --experimental-strip-types
```

---

## Repo layout (Secret Hitler-specific)

```
schemas/secrethitler/    — role/party/phase/event enums, role
                           distribution, power schedule (PowerFor).
models/                  — Dynamo-backed entities: Game, Player,
                           Government, Vote, EnactedPolicy,
                           ExecutiveAction, GameEvent, Passport,
                           PassportEntry. All auto-register with the
                           global models registry.
handlers/game/           — the engine. Pure state machine over the
                           DB, plus a pub-sub Hub and typed Emitter.
                           File roles:
                             base.go       GameHandler + Option wiring
                             engine.go     shared load/save/setPhase
                             start.go      CreateGame/JoinGame/StartGame
                             nominate.go   president picks chancellor
                             vote.go       CastVote + election resolution
                             legislate.go  discard/enact/veto + top-deck
                             execute.go    presidential powers
                             progress.go   ForceProgress, TimerExpired
                             deck.go       shuffle/draw/reshuffle
                             events.go     typed payload structs
                             emitter.go    Envelope / Emitter / SlogEmitter
                             hub.go        MemoryHub (swap for Redis pub-sub
                                           by implementing the Hub iface)
                             clock.go      Clock iface + FakeClock
                             rng.go        RNG iface + SeededRNG
                             config.go     per-phase deadlines
api/                     — Gin HTTP server. Files:
                             server.go, routes.go, middleware.go,
                             auth.go (Nop/Static/Clerk),
                             game_handlers.go (REST),
                             stream_handlers.go (SSE)
cmd/api/                 — minimal main that wires Dynamo + Clerk-or-Nop
                           auth + the Gin server. Deliberately avoids
                           importing the full `connectors` package
                           because connectors/livekit/ has broken deps.
docs/openapi.yaml        — OpenAPI 3.0.3 spec for every endpoint and
                           every event payload.
web/                     — React + TypeScript client (Vite).
                             src/types.ts   — mirrors docs/openapi.yaml
                             src/store.ts   — pure reducer (REST snapshot
                                              + SSE envelope → state)
                             src/api.ts     — typed REST client
                             src/stream.ts  — EventSource wrapper
                             src/App.tsx    — login/lobby/game shell
                             src/store.test.ts — node --test reducer tests
.env.example, web/.env.example
```

---

## Architecture notes (Secret Hitler-specific)

### Phase machine

```
lobby
 └─ StartGame ──▶ nomination ──▶ election ──▶ legislative_president
                  (pres picks)   (all vote)    └─ discard ──▶ legislative_chancellor
                                                 (pres)        └─ enact ──▶
                                                                  { nomination | executive_action | game_over }
                                                                 or veto_requested (if VetoUnlocked)
                                                                                   └─ accept → nomination (tracker++)
                                                                                   └─ reject → legislative_chancellor
```

### Emitter / audience model

Every state-changing engine call produces one or more `Envelope`s:

- `audience.scope == "broadcast"`: everyone in the game, including the
  board device.
- `audience.scope == "player"`: a single recipient's device.

Broadcast carries public info (who voted, what passed). Whispers carry
secrets (role reveal, drawn policies, investigation result, policy peek).
The web `store.ts` only populates its private `me` slice from whispers
whose `audience.playerId` matches the local player.

### Progression triggers

Three ways a phase can advance:

1. Action: players/president call the normal endpoints.
2. All-voted: inside `CastVote`, when every alive player has voted we
   resolve inline with `ReasonAllVoted`.
3. Timeout / Host force: `TimerExpired` (deadline-gated) and
   `ForceProgress` (host-only) both funnel into `progress.advance()`,
   which dispatches per-phase defaults. Each carries a
   `ProgressReason` through `ctxWithReason`/`reasonFromCtx` so
   downstream `setPhase` calls stay labelled correctly.

### Hub / streams

The `Hub` is an interface; the only implementation today is
`MemoryHub` (single process). To go multi-process, implement the same
interface on top of Redis pub-sub and pass it via `api.Options.Hub`.
The Gin SSE handlers at `api/stream_handlers.go` just subscribe and
forward envelopes.

---

## What's done (verified)

- Data models and global registry; `go test ./models/...` green.
- `schemas/secrethitler` enums + `PowerFor` + `RoleDistribution` +
  `PartyFor` helpers.
- Full game engine with
  - random initial president,
  - rules-correct eligibility/term-limit checks,
  - veto flow (propose → accept/reject),
  - every executive power (investigate, special election, policy peek,
    execution) with correct reveals / whispers,
  - election tracker + top-deck,
  - four win conditions.
- **Action matrix test** at `handlers/game/action_matrix_test.go`:
  360 games across player counts 5-10, 6 strategies, 10 seeds;
  asserts every applicable power fires, all 18 event types emit, both
  veto paths terminate, `ForceProgress` and `TimerExpired` produce
  labelled transitions.
- Gin API + SSE streams, REST handlers with auth + phase/role gates,
  role-scrubbing on the public snapshot.
- OpenAPI 3.0.3 spec at `docs/openapi.yaml`.
- React + TypeScript client with typed reducer, REST client, SSE
  subscriber, 8 reducer tests passing under `node --test`.
- `cmd/api/main.go` entrypoint that loads `.env`, wires only the
  database + auth (sidestepping the broken `connectors/livekit`).
- `.env.example` for both backend and web.

Engine bugs found and fixed along the way (useful context if regressions
pop up):

- Top-deck double-rotation: `onElectionFailed` was rotating after
  `topDeck` already rotated via `applyEnactedPolicy`. Fixed by
  returning early in the top-deck branch.
- Election tracker was reset on every elected government. Per the
  rulebook it only resets when a policy is actually *enacted*
  (elected-then-vetoed should not reset). Moved the reset to
  `applyEnactedPolicy`.

---

## Open items / known issues / next steps

1. **CORS**. The Gin server has no CORS middleware. In dev this is
   fine via Vite's `/api` proxy, but the web client currently uses
   `VITE_API_URL = http://localhost:8080` by default, bypassing the
   proxy and triggering CORS errors in the browser. Two fixes needed
   together:
   - Add a CORS middleware in `api/server.go` (allow-list via
     `api.Options.AllowedOrigins`, default `*` in dev).
   - Default the web client's `BASE_URL` to `""` so requests become
     relative and the Vite proxy picks them up.

2. **/docs endpoint**. `docs/openapi.yaml` is checked in but not
   served. Next step: `GET /openapi.yaml` via `embed` and `GET /docs`
   rendering Redoc or Swagger UI, roughly 30 lines.

3. **Lobby UX**. `web/src/App.tsx` `Lobby` component disables both
   buttons until the user fills BOTH join code and display name.
   Hosting should either autogenerate a 5-letter join code or relax
   the disable condition; players still need both.

4. **User ↔ Clerk stitching**. The `models.User` record exists and
   `handlers/user` has `GetByAuthID` / `CreateUser`, but
   `api/middleware.go:requireAuth` stashes the raw Clerk external ID
   as the user key. That means `Game.HostUserID`, `Player.UserID`,
   and the Passport don't use our internal ULID. Fix: in
   `requireAuth`, after verifying the token look up / create a `User`
   row and stash the internal `UserID` on the context. Also add
   `GET /api/v1/me` and `GET /api/v1/passports/me`.

5. **Passport handlers**. `models.Passport` and `PassportEntry`
   exist but nothing writes to them. At `endGame`, iterate players
   and call `Passport.RecordGame(...)` + `NewPassportEntry(...)` and
   persist.

6. **livekit build break**. `connectors/livekit/client.go` imports
   `github.com/livekit/…` packages that aren't in `go.mod`. Because
   `connectors/connectors.go` wires every connector, `go build ./...`
   fails at the root. Options: `go get` the missing modules; or
   gate the livekit connector behind a build tag. Every package
   Secret Hitler uses builds clean individually today.

7. **Redis-backed Hub**. For multi-instance deployment,
   implement `handlers/game/hub.go:Hub` on top of Redis pub-sub.
   `HubEmitter` already adapts any Hub to the engine's Emitter;
   no engine changes needed.

8. **Auth on SSE**. `EventSource` can't set custom headers, so
   `api/stream.ts` passes the token as `?token=` and
   `middleware.go:extractBearerToken` accepts it via the `X-Auth-Token`
   header but not the query string today. Either accept `?token=` on
   the server or switch the SSE endpoints to fetch-based streaming
   (which supports headers).

---

## Commit / branch conventions

- Single feature branch: `claude/secret-hitler-data-models-CiDqk`.
- Commit messages: imperative subject under ~70 chars, then wrapped
  body explaining *why* the change exists. Check `git log --oneline`
  for examples.
- The stop hook at `~/.claude/stop-hook-git-check.sh` enforces that
  untracked files are committed before ending a session.
- Pre-existing repo gitignore covers `.env` at the root; the web/
  gitignore covers `.env`, `.env.local`, `.env.*.local`, plus any
  stray compiled `.js` that slip out of `src/` when `noEmit` is
  toggled off.

---

## Foundation framework background (unchanged from original)

### Core Patterns

**Connectors** (`connectors/`): Interface-based adapters for external services. Each connector type has:
- An interface definition (e.g., `DatabaseConnector`)
- A default implementation (e.g., `DefaultDatabaseConnector`)
- Optional lifecycle hooks (`Init()`, `Close()`)

**Dependencies** (`dependencies/`): Assembles all connectors and handlers into a single `Dependencies` struct that embeds both `Connectors` and `Handlers`. This is the main DI container passed through the application.

**Models** (`models/`): DynamoDB-backed data models implementing the `Model` interface. Models define their own key structure (PK/SK/GSI) and are auto-registered with a global registry via `models.Register()`. The registry enables `models.Lookup(pk, sk)` to instantiate the correct model type from raw DynamoDB items.

**Handlers** (`handlers/`): Business logic that orchestrates connectors. Handlers receive the `Connectors` struct and expose domain operations.

**Schemas** (`schemas/`): Data contracts and constants shared across packages (query filters, relationship types, storage options).

### Connector Types

- `authentication/` - Clerk-based auth (token verification, user creation)
- `cache/` - Redis and in-memory caching
- `channel/` - SMS via Twilio
- `database/` - DynamoDB operations (Get, Upsert, Query)
- `embedding/` - OpenAI text embeddings
- `feature/` - LaunchDarkly feature flags
- `graphdb/` - Neo4j graph relationships
- `notification/` - Slack messaging
- `secret/` - AWS Secrets Manager
- `storage/` - S3 file storage
- `vector/` - Qdrant vector search

### DynamoDB Model Pattern

Models use composite keys with prefixes for type identification:
```go
type User struct {
    models.DynamoMetadata
    ID    string
    Email string
}

func (u *User) TypeIdentifier() (string, string) {
    return "USER#", "USER#"
}
```

Register models at init time:
```go
func init() {
    models.Register("USER#", "USER#", func() models.Model { return &User{} })
}
```

### Local Development

Docker Compose provides:
- Redis (6379)
- DynamoDB Local (8000) with Admin UI (8001)
- Neo4j (7474/7687)
- Qdrant (6333/6334)

Environment variables are managed via `.envrc` (direnv).
