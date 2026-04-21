// SSE hook — connects directly to the Go server (bypasses the Next
// dev proxy, which buffers long-lived responses) and self-heals when
// the token rotates or the network flaps.
//
// Why the manual reconnect loop instead of letting EventSource retry
// natively: a Clerk JWT has a ~60s TTL. Native EventSource reconnect
// reuses the exact URL it was given, meaning the stale token gets
// re-sent and the server returns 401 forever. We intercept the error
// event, close the stream, pull a fresh token from the supplier, and
// open a new EventSource with exponential backoff.

"use client";

import { useEffect, useRef } from "react";
import type { Envelope } from "@replicant/schema";
import { EventType as EventTypeSchema } from "@replicant/schema";
import { API_ORIGIN } from "./env";

/** token can be a fixed string (legacy callers) or an async supplier
 *  that returns the current bearer. Supplier is called fresh on each
 *  connection attempt so rotated JWTs take effect automatically. */
export type StreamToken = string | (() => string | Promise<string>) | null;

export interface UseStreamOptions {
  gameId: string | null;
  token: StreamToken;
  onEnvelope: (env: Envelope) => void;
  onError?: (err: Event) => void;
  onOpen?: () => void;
}

export function useStream({ gameId, token, onEnvelope, onError, onOpen }: UseStreamOptions) {
  const cbRef = useRef(onEnvelope);
  cbRef.current = onEnvelope;
  const errRef = useRef(onError);
  errRef.current = onError;
  const openRef = useRef(onOpen);
  openRef.current = onOpen;
  const tokenRef = useRef<StreamToken>(token);
  tokenRef.current = token;

  useEffect(() => {
    if (!gameId) return;
    let cancelled = false;
    let es: EventSource | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;

    const connect = async () => {
      if (cancelled) return;
      const tok = tokenRef.current;
      const resolved = typeof tok === "function" ? await tok() : tok;
      if (!resolved || cancelled) return;
      const base =
        API_ORIGIN ||
        (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");
      const url = new URL(`/api/v1/games/${gameId}/stream/player`, base);
      url.searchParams.set("token", resolved);

      // eslint-disable-next-line no-console
      console.debug("[stream] connecting", { gameId, attempt });
      es = new EventSource(url.toString(), { withCredentials: true });

      es.addEventListener("open", () => {
        // eslint-disable-next-line no-console
        console.debug("[stream] open", { gameId });
        attempt = 0;
        openRef.current?.();
      });

      es.addEventListener("error", (e: Event) => {
        // eslint-disable-next-line no-console
        console.warn("[stream] error", { gameId, readyState: es?.readyState });
        errRef.current?.(e);
        if (cancelled) return;
        // Close so native reconnect doesn't retry with the stale
        // token; we'll rebuild with a fresh one.
        es?.close();
        es = null;
        attempt += 1;
        // 1s → 2s → 4s → … capped at 15s. The sleep lets Clerk rotate
        // if we raced it, and avoids hammering the server on a
        // persistent failure (e.g. the whole backend is down).
        const delay = Math.min(1000 * 2 ** Math.max(0, attempt - 1), 15000);
        reconnectTimer = setTimeout(() => {
          void connect();
        }, delay);
      });

      for (const t of [...EventTypeSchema.options, "message"]) {
        es.addEventListener(t, (raw: Event) => {
          const ev = raw as MessageEvent<string>;
          if (!ev.data) return;
          try {
            const env = JSON.parse(ev.data) as Envelope;
            // eslint-disable-next-line no-console
            console.debug("[stream] recv", env.event?.type, env);
            cbRef.current(env);
          } catch (err) {
            // eslint-disable-next-line no-console
            console.error("[stream] bad envelope", err, ev.data);
          }
        });
      }
    };

    void connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (es) es.close();
    };
    // token deliberately omitted from deps: the supplier is read
    // through tokenRef on every connect, and pinning on token would
    // tear down the live connection whenever the auth callback's
    // identity changes (which happens on sign-in/out). String-valued
    // tokens still propagate through tokenRef — they just won't
    // trigger a proactive reconnect until the next natural retry.
  }, [gameId]); // eslint-disable-line react-hooks/exhaustive-deps
}
