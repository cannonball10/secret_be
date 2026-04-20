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

function randomId(): string {
  const bytes = new Uint8Array(8);
  (globalThis.crypto ?? window.crypto).getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
