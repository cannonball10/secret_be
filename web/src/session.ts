// session.ts — localStorage-backed device session.
//
// One object, one key, narrow API. The value holds just what we need
// to re-open a live game after a refresh: the bearer token, the
// gameId, and which role the device is playing. On boot, App.tsx
// reads this, verifies the game is still live via GET /games/:id, and
// either restores the route or wipes the session.
//
// We intentionally do NOT cache per-game state here — the server is
// the source of truth, and GET /games/:id plus the SSE stream will
// rehydrate everything. Keeping this tiny means fewer places for stale
// data to hide.

const KEY = "sh:session";
const TTL_MS = 12 * 60 * 60 * 1000; // 12 hours — long enough for a party session, short enough to be hygienic

export interface Session {
  token: string;
  gameId: string;
  role: "board" | "player";
  savedAt: number;
}

export function loadSession(): Session | null {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Session;
    if (!parsed?.token || !parsed?.gameId || !parsed?.role) return null;
    if (Date.now() - parsed.savedAt > TTL_MS) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function saveSession(s: Omit<Session, "savedAt">): void {
  try {
    const full: Session = { ...s, savedAt: Date.now() };
    localStorage.setItem(KEY, JSON.stringify(full));
  } catch {
    // Quota / private mode / SSR — fail quietly. Worst case the user
    // re-logs in on next visit.
  }
}

export function clearSession(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    // See above.
  }
}
