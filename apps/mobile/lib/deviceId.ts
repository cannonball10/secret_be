// Per-device user token, persisted in localStorage. NopAuth on the
// Go server maps this to the caller's userID, so every call from
// this phone is recognised as the same player.

const KEY = "replicant:deviceId";

export function deviceId(): string {
  if (typeof window === "undefined") return "ssr";
  try {
    let v = window.localStorage.getItem(KEY);
    if (!v) {
      v = `player-${randomId()}`;
      window.localStorage.setItem(KEY, v);
    }
    return v;
  } catch {
    return `player-${randomId()}`;
  }
}

/** Force a brand-new deviceId. Used by /join?asNew=1 so a single
 *  browser profile can seat as multiple players during local QA. */
export function resetDeviceId(): string {
  const next = `player-${randomId()}`;
  if (typeof window === "undefined") return next;
  try {
    window.localStorage.setItem(KEY, next);
  } catch {
    // ignore
  }
  return next;
}

function randomId(): string {
  const bytes = new Uint8Array(8);
  (globalThis.crypto ?? window.crypto).getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
