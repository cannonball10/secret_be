// session.ts — mobile's local record of which game the player is in,
// their own playerId, and the token they authenticated with. Survives
// refreshes so a phone reload doesn't drop the player out of the game.

import type { Role, Party, TeammateInfo } from "@replicant/schema";

export interface MobileSession {
  token: string;
  gameId: string;
  joinCode: string;
  playerId: string;
  displayName: string;
  savedAt: number;
  /** Populated once roles_assigned arrives. */
  role?: Role;
  party?: Party;
  teammates?: TeammateInfo[];
  /**
   * True once the player has tapped "I ACCEPT" on the role-reveal
   * screen. Persisted so a refresh lands on the concealed view
   * rather than spoiling the role again.
   */
  accepted?: boolean;
}

const KEY = "replicant:mobile:session";
const TTL_MS = 12 * 60 * 60 * 1000;

export function loadSession(): MobileSession | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as MobileSession;
    if (!parsed?.token || !parsed?.gameId || !parsed?.playerId) return null;
    if (Date.now() - parsed.savedAt > TTL_MS) return null;
    return parsed;
  } catch {
    return null;
  }
}

export function saveSession(s: Omit<MobileSession, "savedAt">): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(KEY, JSON.stringify({ ...s, savedAt: Date.now() }));
  } catch {
    // ignore
  }
}

/** Merge partial updates (e.g. role from whisper) into the stored session. */
export function patchSession(patch: Partial<MobileSession>): MobileSession | null {
  const curr = loadSession();
  if (!curr) return null;
  const next = { ...curr, ...patch, savedAt: Date.now() };
  saveSession(next);
  return next;
}

export function clearSession(): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(KEY);
  } catch {
    // ignore
  }
}
