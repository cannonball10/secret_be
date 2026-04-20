// session.ts — localStorage-backed record of the active game so a
// refresh lands back on the live lobby / game screen instead of the
// title. Shared shape across host + mobile, but host only ever stores
// role "board" sessions.

export interface HostSession {
  token: string;
  gameId: string;
  role: "board";
  joinCode: string;
  savedAt: number;
}

const KEY = "replicant:host:session";
const TTL_MS = 12 * 60 * 60 * 1000;

export function loadSession(): HostSession | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as HostSession;
    if (!parsed?.token || !parsed?.gameId || !parsed?.joinCode) return null;
    if (Date.now() - parsed.savedAt > TTL_MS) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function saveSession(s: Omit<HostSession, "savedAt">): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(KEY, JSON.stringify({ ...s, savedAt: Date.now() }));
  } catch {
    // quota / private mode — ignore
  }
}

export function clearSession(): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(KEY);
  } catch {
    // ignore
  }
}
