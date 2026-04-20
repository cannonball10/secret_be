// Host-side volume settings, persisted in localStorage and read by
// every audio-playing component (currently just the narrator overlay;
// background music lands here once it's wired).
//
// All values are 0-1 floats matching HTMLMediaElement.volume and Web
// Audio GainNode.gain.value. Subscribers register via subscribe() and
// get called whenever the settings change — no extra state library.

"use client";

export interface VolumeSettings {
  master: number;
  narrator: number;
  music: number;
}

export const DEFAULT_VOLUME: VolumeSettings = {
  master: 0.9,
  narrator: 1.0,
  music: 0.5,
};

const STORAGE_KEY = "replicant.host.volume.v1";

function clamp(v: number): number {
  if (!Number.isFinite(v)) return 0;
  if (v < 0) return 0;
  if (v > 1) return 1;
  return v;
}

function readFromStorage(): VolumeSettings {
  if (typeof window === "undefined") return DEFAULT_VOLUME;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_VOLUME;
    const parsed = JSON.parse(raw) as Partial<VolumeSettings>;
    return {
      master: clamp(parsed.master ?? DEFAULT_VOLUME.master),
      narrator: clamp(parsed.narrator ?? DEFAULT_VOLUME.narrator),
      music: clamp(parsed.music ?? DEFAULT_VOLUME.music),
    };
  } catch {
    return DEFAULT_VOLUME;
  }
}

function writeToStorage(v: VolumeSettings) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(v));
  } catch {
    /* quota full or private-mode; drop silently */
  }
}

let cached: VolumeSettings | null = null;
const subscribers = new Set<(v: VolumeSettings) => void>();

/** Read the current volume settings. Initialises from localStorage on
 *  first call; subsequent calls return the cached in-memory copy. */
export function getVolume(): VolumeSettings {
  if (cached === null) cached = readFromStorage();
  return cached;
}

/** Mutate volume settings and broadcast to every subscriber. Partial
 *  updates are merged on top of the current settings. */
export function setVolume(patch: Partial<VolumeSettings>) {
  const next: VolumeSettings = { ...getVolume() };
  if (patch.master !== undefined) next.master = clamp(patch.master);
  if (patch.narrator !== undefined) next.narrator = clamp(patch.narrator);
  if (patch.music !== undefined) next.music = clamp(patch.music);
  cached = next;
  writeToStorage(next);
  subscribers.forEach((fn) => fn(next));
}

/** Subscribe to volume changes. Returns an unsubscribe fn. */
export function subscribeVolume(fn: (v: VolumeSettings) => void): () => void {
  subscribers.add(fn);
  // Fire once with the current value so subscribers can initialise
  // without a separate getVolume() call.
  fn(getVolume());
  return () => {
    subscribers.delete(fn);
  };
}

/** Compute the effective volume for a channel — master × channel,
 *  in [0, 1]. */
export function effectiveVolume(channel: keyof Omit<VolumeSettings, "master">): number {
  const v = getVolume();
  return clamp(v.master * v[channel]);
}
