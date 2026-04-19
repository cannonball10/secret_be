# secret-hitler-web

Dead-simple React + TypeScript client for the Secret Hitler backend.

## Layout

```
src/
  types.ts    — enums + payload types mirroring docs/openapi.yaml
  store.ts    — pure reducer: REST snapshots + SSE envelopes → state
  api.ts      — REST client (lobby, host, player actions)
  stream.ts   — SSE subscriber, one per device
  App.tsx     — login / lobby / game shell, routes the two device roles
  main.tsx    — entry point
```

The state reducer is pure and doesn't touch the network. Everything the
UI renders is a function of `GameState`:

- **Public slice** — `game`, `players`, `currentGovernment`, policy
  counts, election tracker, event log.
- **Private slice (`me`)** — role, party, teammates, any policies the
  server has whispered (drawn, chancellor options, peeked), last
  investigation result. Populated only by `audience.scope === "player"`
  envelopes addressed to the local player.

Swap the ad-hoc login input for a real identity provider (Clerk, etc.)
by replacing `Login` in `App.tsx`; the token is just passed through to
`ApiClient` and the SSE URL.

## Running

```bash
npm install
npm run dev
```

The dev server proxies `/api` and `/healthz` to `localhost:8080` (see
`vite.config.ts`), so `go run ./...` in the repo root is enough.

## Typecheck

```bash
npm run typecheck
```

## Device roles

- **Board device**: the host's big screen. Opens
  `GET /api/v1/games/:id/stream/board`, shows every broadcast
  envelope, never receives whispers. Exposes host-only buttons
  (`Start game`, `Force progress`).
- **Player device**: each player's phone. Opens
  `GET /api/v1/games/:id/stream/player`, receives broadcasts plus
  whispers addressed to its player record. The `PlayerControls`
  component shows exactly the action the current phase permits for
  the logged-in player.
