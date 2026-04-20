// apps/host/lib/api.ts — typed REST client for the host device.
//
// All requests go to same-origin /api/... paths, which Next rewrites
// to the Go server (see next.config.js). The Bearer token is supplied
// per-request by the caller — we don't stash it here because different
// screens may have different identities (e.g. anonymous boot vs.
// a logged-in host).

import type {
  CableLeakedPayload,
  Game,
  NarratorSpeakPayload,
  Player,
  RulesConfig,
} from "@replicant/schema";
import { API_ORIGIN } from "./env";

export interface ApiClientOptions {
  token: string;
  baseUrl?: string;
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly body: unknown,
    message: string,
  ) {
    super(message);
  }
}

export class HostApi {
  constructor(private opts: ApiClientOptions) {}

  /** Create a new session. Empty displayName ⇒ board-only (no Player). */
  createGame(displayName = ""): Promise<{ game: Game; player: Player | null }> {
    return this.post("/api/v1/games", { joinCode: "", displayName });
  }

  getGame(gameId: string): Promise<{ game: Game; players: Player[] }> {
    return this.get(`/api/v1/games/${gameId}`);
  }

  /** Fetch the persisted cable leak for the game's current round.
   *  Used on boot to recover the leak banner when the SSE stream
   *  opened after the cable_leaked envelope already fired. */
  currentLeak(gameId: string): Promise<{ leak: CableLeakedPayload | null }> {
    return this.get(`/api/v1/games/${gameId}/current-leak`);
  }

  startGame(gameId: string): Promise<{ game: Game }> {
    return this.post(`/api/v1/games/${gameId}/host/start`);
  }

  forceProgress(gameId: string): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/host/force-progress`);
  }

  /** Replace a lobby's RulesConfig. Host-only, lobby-only on the server. */
  updateRules(gameId: string, rules: RulesConfig): Promise<{ game: Game }> {
    return this.put<{ game: Game }>(`/api/v1/games/${gameId}/host/rules`, { rules });
  }

  /**
   * Generate a narrator utterance for the given cue kind and broadcast
   * it. Custom cues (`cue: "custom"`) take `text` verbatim and skip
   * the LLM. Server returns the payload synchronously and also fans
   * it out as a `narrator_speak` envelope to every subscriber.
   */
  narrate(
    gameId: string,
    cue: string,
    opts: { text?: string; vars?: Record<string, string> } = {},
  ): Promise<NarratorSpeakPayload> {
    return this.post(`/api/v1/games/${gameId}/host/narrate`, {
      cue,
      text: opts.text ?? "",
      vars: opts.vars ?? {},
    });
  }

  private get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  private post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  private put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("PUT", path, body);
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const url = (this.opts.baseUrl ?? API_ORIGIN) + path;
    const res = await fetch(url, {
      method,
      headers: {
        Authorization: `Bearer ${this.opts.token}`,
        "Content-Type": "application/json",
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    const text = await res.text();
    const parsed = text ? safeJson(text) : null;
    if (!res.ok) {
      const msg =
        (parsed && typeof parsed === "object" && "error" in parsed && typeof (parsed as { error: unknown }).error === "string"
          ? (parsed as { error: string }).error
          : res.statusText) || `HTTP ${res.status}`;
      throw new ApiError(res.status, parsed, msg);
    }
    return parsed as T;
  }
}

function safeJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}
