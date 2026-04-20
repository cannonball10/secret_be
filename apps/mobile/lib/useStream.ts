// Mirror of apps/host/lib/useStream.ts — connect directly to the Go
// server and keep the callback behind a ref so state-derived handlers
// don't retear the EventSource on every render.

"use client";

import { useEffect, useRef } from "react";
import type { Envelope } from "@replicant/schema";
import { EventType as EventTypeSchema } from "@replicant/schema";
import { API_ORIGIN } from "./env";

export interface UseStreamOptions {
  gameId: string | null;
  token: string | null;
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

  useEffect(() => {
    if (!gameId || !token) return;
    const base =
      API_ORIGIN ||
      (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");
    const url = new URL(`/api/v1/games/${gameId}/stream/player`, base);
    url.searchParams.set("token", token);

    // eslint-disable-next-line no-console
    console.debug("[stream] connecting", { gameId, url: url.toString() });
    const es = new EventSource(url.toString(), { withCredentials: true });

    const onOpenH = () => {
      // eslint-disable-next-line no-console
      console.debug("[stream] open", { gameId });
      openRef.current?.();
    };
    const onErrH = (e: Event) => {
      // eslint-disable-next-line no-console
      console.warn("[stream] error", { gameId, readyState: es.readyState });
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
  }, [gameId, token]);
}
