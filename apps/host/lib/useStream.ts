// useStream.ts — React hook for SSE consumption.
//
// Subscribes to the backend's /stream/{role} endpoint, which emits an
// `event: <eventType>` per frame. We bind listeners for every known
// EventType so envelopes dispatch through a single handler without
// needing to parse the event name ourselves.
//
// Self-heals when the token rotates or the network flaps. See
// apps/mobile/lib/useStream.ts for the full rationale — the two
// hooks are intentionally kept in sync.

"use client";

import { useEffect, useRef } from "react";
import type { Envelope } from "@replicant/schema";
import { EventType as EventTypeSchema } from "@replicant/schema";
import { API_ORIGIN } from "./env";

export type StreamToken = string | (() => string | Promise<string>) | null;

export interface UseStreamOptions {
  gameId: string | null;
  role: "board" | "player";
  token: StreamToken;
  onEnvelope: (env: Envelope) => void;
  onError?: (err: Event) => void;
  onOpen?: () => void;
}

export function useStream({ gameId, role, token, onEnvelope, onError, onOpen }: UseStreamOptions) {
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
      // Connect DIRECTLY to the Go server (not through Next rewrites) —
      // Next dev mode buffers long-lived responses through its proxy,
      // which breaks SSE. CORS is enabled on the server for this origin.
      const base =
        API_ORIGIN ||
        (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");
      const url = new URL(`/api/v1/games/${gameId}/stream/${role}`, base);
      url.searchParams.set("token", resolved);

      // eslint-disable-next-line no-console
      console.debug("[stream] connecting", { gameId, role, attempt });
      es = new EventSource(url.toString(), { withCredentials: true });

      es.addEventListener("open", () => {
        // eslint-disable-next-line no-console
        console.debug("[stream] open", { gameId, role });
        attempt = 0;
        openRef.current?.();
      });

      es.addEventListener("error", (e: Event) => {
        // eslint-disable-next-line no-console
        console.warn("[stream] error", { gameId, role, readyState: es?.readyState });
        errRef.current?.(e);
        if (cancelled) return;
        es?.close();
        es = null;
        attempt += 1;
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
    // token intentionally omitted — see mobile useStream for the
    // full explanation; the supplier is read through tokenRef on
    // every connect attempt, so we don't need to re-run this effect
    // when the callback identity changes.
  }, [gameId, role]); // eslint-disable-line react-hooks/exhaustive-deps
}
