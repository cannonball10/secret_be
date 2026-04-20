// useStream.ts — React hook for SSE consumption.
//
// Subscribes to the backend's /stream/{role} endpoint, which emits an
// `event: <eventType>` per frame. We bind listeners for every known
// EventType so envelopes dispatch through a single handler without
// needing to parse the event name ourselves.

"use client";

import { useEffect, useRef } from "react";
import type { Envelope } from "@replicant/schema";
import { EventType as EventTypeSchema } from "@replicant/schema";
import { API_ORIGIN } from "./env";

export interface UseStreamOptions {
  gameId: string | null;
  role: "board" | "player";
  token: string | null;
  onEnvelope: (env: Envelope) => void;
  onError?: (err: Event) => void;
  onOpen?: () => void;
}

// useStream subscribes to the backend SSE stream. The callback stays
// behind a ref so consumers can depend on local state without retearing
// the connection on every render.
export function useStream({ gameId, role, token, onEnvelope, onError, onOpen }: UseStreamOptions) {
  const cbRef = useRef(onEnvelope);
  cbRef.current = onEnvelope;
  const errRef = useRef(onError);
  errRef.current = onError;
  const openRef = useRef(onOpen);
  openRef.current = onOpen;

  useEffect(() => {
    if (!gameId || !token) return;
    // Connect DIRECTLY to the Go server (not through Next rewrites) —
    // Next dev mode buffers long-lived responses through its proxy,
    // which breaks SSE. CORS is enabled on the server for this origin.
    const base =
      API_ORIGIN ||
      (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");
    const url = new URL(`/api/v1/games/${gameId}/stream/${role}`, base);
    url.searchParams.set("token", token);

    // eslint-disable-next-line no-console
    console.debug("[stream] connecting", { gameId, role, url: url.toString() });
    const es = new EventSource(url.toString(), { withCredentials: true });

    const onOpenH = () => {
      // eslint-disable-next-line no-console
      console.debug("[stream] open", { gameId, role });
      openRef.current?.();
    };
    const onErrH = (e: Event) => {
      // eslint-disable-next-line no-console
      console.warn("[stream] error", { gameId, role, readyState: es.readyState });
      errRef.current?.(e);
    };
    es.addEventListener("open", onOpenH);
    es.addEventListener("error", onErrH);

    const handlers: Array<[string, (e: Event) => void]> = [];
    for (const t of [...EventTypeSchema.options, "message"]) {
      const h = (raw: Event) => {
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
      };
      es.addEventListener(t, h);
      handlers.push([t, h]);
    }

    return () => {
      es.removeEventListener("open", onOpenH);
      es.removeEventListener("error", onErrH);
      for (const [t, h] of handlers) es.removeEventListener(t, h);
      es.close();
    };
  }, [gameId, role, token]);
}
