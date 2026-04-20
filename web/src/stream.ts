// streamEnvelopes opens an SSE connection to one of the two backend
// streams and invokes onEnvelope for every frame. Returns a close
// handle; call it to unsubscribe (e.g. on React unmount).
//
// Board devices use "board" role; player devices use "player" role.
// The backend enforces that only the host can open a board stream and
// that a player stream maps to whatever player the bearer token
// resolves to.

import type { Envelope } from "./types";

export type StreamRole = "board" | "player";

export interface StreamHandle {
  close(): void;
}

export interface StreamOptions {
  baseUrl: string;
  gameId: string;
  role: StreamRole;
  token: string;
  onEnvelope: (env: Envelope) => void;
  onOpen?: () => void;
  onError?: (err: Event) => void;
}

// Opens the SSE stream. EventSource doesn't accept custom headers, so
// we embed the token as a query-string parameter the backend also
// accepts (via X-Auth-Token fallback, wired server-side). For now we
// rely on the server accepting the bearer token on the query string;
// swap this for a native WebSocket or fetch-based SSE in production if
// strict header-only auth is required.
export function streamEnvelopes(opts: StreamOptions): StreamHandle {
  // baseUrl is "" in dev (the Vite proxy forwards /api to :8080), so we
  // can't hand the path to `new URL(path)` directly — it requires either
  // an absolute URL or a base. Resolving against window.location.origin
  // keeps the request same-origin and lets the proxy do its job.
  const base = opts.baseUrl || window.location.origin;
  const url = new URL(
    `/api/v1/games/${opts.gameId}/stream/${opts.role}`,
    base,
  );
  // Most browsers don't send custom headers on EventSource, so we pass
  // the token as a query param that the server accepts (see
  // api/middleware.go: extractBearerToken via X-Auth-Token or the
  // bearer param). Production deployments typically front the SSE
  // endpoint with a reverse proxy that terminates TLS and forwards
  // cookies; for development the query param is fine.
  url.searchParams.set("token", opts.token);

  const src = new EventSource(url.toString(), { withCredentials: true });

  if (opts.onOpen) src.addEventListener("open", () => opts.onOpen?.());
  if (opts.onError) src.addEventListener("error", (e) => opts.onError?.(e));

  // The backend emits SSE frames with `event: <eventType>` so clients
  // could bind per-type listeners, but we prefer a single onmessage-ish
  // handler that fans out into the reducer. Listening on every known
  // event type keeps things declarative.
  const eventTypes = [
    "game_created",
    "player_joined",
    "player_left",
    "game_started",
    "roles_assigned",
    "chancellor_nominated",
    "vote_cast",
    "election_result",
    "policies_drawn",
    "president_discarded",
    "chancellor_enacted",
    "veto_proposed",
    "veto_resolved",
    "executive_action",
    "election_tracker_advanced",
    "top_deck_enacted",
    "deck_reshuffled",
    "player_executed",
    "game_ended",
    "message", // default EventSource type
  ];
  for (const t of eventTypes) {
    src.addEventListener(t, (raw) => {
      const ev = raw as MessageEvent<string>;
      if (!ev.data) return;
      try {
        const parsed = JSON.parse(ev.data) as Envelope;
        opts.onEnvelope(parsed);
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error("stream: bad envelope", err, ev.data);
      }
    });
  }

  return {
    close() {
      src.close();
    },
  };
}
