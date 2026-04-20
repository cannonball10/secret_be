// Thin REST client for the backend. Every call carries the caller's
// bearer token in Authorization. Errors are surfaced as rejected
// Promises with the parsed JSON body (if any).

import type { Game, Government, Player, VoteChoice } from "./types";

export interface ApiClientOptions {
  baseUrl: string; // e.g. "https://api.example.com"
  token: string; // bearer token (ClerkAuth in prod, StaticAuth in dev)
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

export class ApiClient {
  constructor(private opts: ApiClientOptions) {}

  // ─── lobby ────────────────────────────────────────────────────

  // createGame posts to /games. Both fields are optional: an empty
  // joinCode asks the server to generate one, and an empty displayName
  // creates a board-only lobby where the host device is a spectator
  // (no Player record). When displayName is empty the server responds
  // with `player: null`.
  createGame(joinCode?: string, displayName?: string) {
    return this.post<{ game: Game; player: Player | null }>("/api/v1/games", {
      joinCode: joinCode ?? "",
      displayName: displayName ?? "",
    });
  }

  // joinGame returns the caller's Player record plus the full seat list
  // (roles scrubbed for everyone except the caller) so the joining
  // device has a complete lobby snapshot immediately — MemoryHub
  // doesn't replay, so any players who joined before the SSE opened
  // would otherwise be invisible forever.
  joinGame(joinCode: string, displayName: string) {
    return this.post<{ game: Game; player: Player; players: Player[] }>(
      "/api/v1/games/join",
      { joinCode, displayName },
    );
  }

  getGame(gameId: string) {
    return this.get<{ game: Game; players: Player[] }>(`/api/v1/games/${gameId}`);
  }

  // ─── host actions (board device) ──────────────────────────────

  startGame(gameId: string) {
    return this.post<{ game: Game }>(`/api/v1/games/${gameId}/host/start`);
  }

  forceProgress(gameId: string) {
    return this.post<{ status: string }>(`/api/v1/games/${gameId}/host/force-progress`);
  }

  timerTick(gameId: string) {
    return this.post<{ status: string }>(`/api/v1/games/${gameId}/host/timer-tick`);
  }

  // ─── player actions ───────────────────────────────────────────

  nominateChancellor(gameId: string, chancellorPlayerId: string) {
    return this.post<{ government: Government }>(
      `/api/v1/games/${gameId}/player/nominate`,
      { chancellorPlayerId },
    );
  }

  castVote(gameId: string, choice: VoteChoice) {
    return this.post<{ status: string }>(`/api/v1/games/${gameId}/player/vote`, {
      choice,
    });
  }

  presidentDiscard(gameId: string, index: number) {
    return this.post<{ status: string }>(
      `/api/v1/games/${gameId}/player/discard`,
      { index },
    );
  }

  chancellorEnact(gameId: string, index: number) {
    return this.post<{ status: string }>(
      `/api/v1/games/${gameId}/player/enact`,
      { index },
    );
  }

  proposeVeto(gameId: string) {
    return this.post<{ status: string }>(`/api/v1/games/${gameId}/player/veto`);
  }

  resolveVeto(gameId: string, accepted: boolean) {
    return this.post<{ status: string }>(
      `/api/v1/games/${gameId}/player/veto/resolve`,
      { accepted },
    );
  }

  executeAction(gameId: string, targetPlayerId: string) {
    return this.post<{ status: string }>(
      `/api/v1/games/${gameId}/player/action`,
      { targetPlayerId },
    );
  }

  // ─── internals ────────────────────────────────────────────────

  private async get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  private async post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(this.opts.baseUrl + path, {
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
