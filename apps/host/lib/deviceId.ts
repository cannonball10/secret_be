// deviceId.ts — per-device identity, persisted in localStorage.
//
// NopAuth on the Go server treats the bearer token as the userID, so
// we need a stable string per device to preserve host ownership of a
// session across refreshes. We generate one once and store it; no
// human-facing login required in dev.

const KEY = "replicant:deviceId";

export function deviceId(): string {
  if (typeof window === "undefined") return "ssr";
  try {
    let v = window.localStorage.getItem(KEY);
    if (!v) {
      v = `host-${randomId()}`;
      window.localStorage.setItem(KEY, v);
    }
    return v;
  } catch {
    // Private mode / storage disabled — fall back to a session-local
    // id. Host ownership won't survive a refresh, but nothing else
    // breaks.
    return `host-${randomId()}`;
  }
}

function randomId(): string {
  const bytes = new Uint8Array(8);
  (globalThis.crypto ?? window.crypto).getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
