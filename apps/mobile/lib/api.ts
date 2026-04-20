// Typed REST client for the mobile/player device. Mirrors
// apps/host/lib/api.ts but exposes only the endpoints a player
// actually needs.

import type { ChatChannel, Game, Government, Player, VoteChoice } from "@replicant/schema";
import { API_ORIGIN } from "./env";

export class ApiError extends Error {
  constructor(public readonly status: number, public readonly body: unknown, message: string) {
    super(message);
  }
}

export class MobileApi {
  constructor(private opts: { token: string; baseUrl?: string }) {}

  joinGame(
    joinCode: string,
    displayName: string,
  ): Promise<{ game: Game; player: Player; players: Player[] }> {
    return this.post("/api/v1/games/join", { joinCode, displayName });
  }

  getGame(gameId: string): Promise<{ game: Game; players: Player[] }> {
    return this.get(`/api/v1/games/${gameId}`);
  }

  // ─── player actions ─────────────────────────────────────────────

  nominateChancellor(gameId: string, chancellorPlayerId: string): Promise<{ government: Government }> {
    return this.post(`/api/v1/games/${gameId}/player/nominate`, { chancellorPlayerId });
  }

  castVote(gameId: string, choice: VoteChoice): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/vote`, { choice });
  }

  presidentDiscard(gameId: string, index: number): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/discard`, { index });
  }

  chancellorEnact(gameId: string, index: number): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/enact`, { index });
  }

  proposeVeto(gameId: string): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/veto`);
  }

  resolveVeto(gameId: string, accepted: boolean): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/veto/resolve`, { accepted });
  }

  executeAction(gameId: string, targetPlayerId: string): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/action`, { targetPlayerId });
  }

  sendChat(gameId: string, channel: ChatChannel, body: string): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/chat`, { channel, body });
  }

  /** Send a direct message to another delegate. Open at any time
   *  during active play — no phase gate server-side. */
  sendDM(
    gameId: string,
    recipientPlayerId: string,
    body: string,
  ): Promise<{ status: string }> {
    return this.post(`/api/v1/games/${gameId}/player/dm`, { recipientPlayerId, body });
  }

  private get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  private post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
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
