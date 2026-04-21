// Typed REST client for the mobile/player device. Mirrors
// apps/host/lib/api.ts but exposes only the endpoints a player
// actually needs.

import type { ChatChannel, ChatMessagePayload, Game, Government, Player, VoteChoice } from "@replicant/schema";
import { API_ORIGIN } from "./env";

/** Tokens may be static (a guest deviceId) or an async Clerk JWT
 *  supplier. Passing a function allows Clerk tokens to refresh
 *  per-request without the caller threading a useEffect. */
export type TokenSupplier = string | (() => string | Promise<string>);

export class ApiError extends Error {
  constructor(public readonly status: number, public readonly body: unknown, message: string) {
    super(message);
  }
}

export class MobileApi {
  constructor(private opts: { token: TokenSupplier; baseUrl?: string }) {}

  /** Fetch the authenticated caller's identity + passport. */
  me(): Promise<{
    user: { userId: string; displayName: string; email: string; authenticationProvider: string };
    passport: unknown;
    provider: string;
  }> {
    return this.get("/api/v1/me");
  }

  /** Merge a guest's passport history onto the signed-in user.
   *  Call this once right after the first successful Clerk sign-in
   *  with the deviceId the caller was using beforehand. */
  linkGuest(deviceId: string): Promise<{ linked: boolean; reason?: string; entriesMoved?: number }> {
    return this.post("/api/v1/me/link", { deviceId });
  }

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

  /** Fetch the caller's persisted DM history so threads survive
   *  mobile reloads. SSE only replays from the moment the stream
   *  opens, so without this reloaded devices would start with empty
   *  threads. Returned messages are in oldest-first order. */
  dmHistory(gameId: string): Promise<{ messages: ChatMessagePayload[] }> {
    return this.get(`/api/v1/games/${gameId}/player/dms`);
  }

  /** Ask the in-game rules helper a natural-language question. */
  ask(gameId: string, question: string): Promise<{ answer: string; faq: string[] }> {
    return this.post(`/api/v1/games/${gameId}/player/ask`, { question });
  }

  /** Fetch the per-player FAQ starter prompts. */
  faq(gameId: string): Promise<{ faq: string[] }> {
    return this.get(`/api/v1/games/${gameId}/player/faq`);
  }

  private get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  private post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  private async resolveToken(): Promise<string> {
    const tok = this.opts.token;
    if (typeof tok === "function") return (await tok()) ?? "";
    return tok ?? "";
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const url = (this.opts.baseUrl ?? API_ORIGIN) + path;
    const token = await this.resolveToken();
    const res = await fetch(url, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
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
