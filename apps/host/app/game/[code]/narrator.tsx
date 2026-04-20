// HostNarrator — overlay that plays the Department's voice on the
// host TV with an 80-bar waveform driven by real audio amplitude and
// a typewriter reveal of the script.
//
// Playback path:
//   · fetch(audioUrl) → ArrayBuffer (works for data: URLs and http)
//   · AudioContext.decodeAudioData → AudioBuffer
//   · AudioBufferSourceNode → AnalyserNode → AudioContext.destination
//   · requestAnimationFrame samples byte-frequency-data into 80 bars
//
// Autoplay: browsers refuse play() if the last user gesture is older
// than ~1-5s. Our narrator generation takes 3-10s, so auto-start
// almost always fails. We attempt autoplay anyway — on failure we
// render a "Press to begin" button whose click preserves the user
// gesture and starts the graph from there.

"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { rpColors } from "@replicant/tokens";
import type { NarratorSpeakPayload } from "@replicant/schema";

const BAR_COUNT = 80;
const CYAN_BARS = 60;

interface Props {
  payload: NarratorSpeakPayload;
  onComplete: () => void;
  /** Optional: force-dismiss (used by the SKIP button). */
  onSkip?: () => void;
}

type PlayState = "idle" | "loading" | "playing" | "ended" | "awaiting-gesture" | "error";

export function HostNarrator({ payload, onComplete, onSkip }: Props) {
  const [state, setState] = useState<PlayState>("idle");
  const [err, setErr] = useState<string | null>(null);

  // Waveform bar heights. Initial gentle sine so the visual isn't
  // flat before the AnalyserNode starts producing real data.
  const [bars, setBars] = useState<number[]>(() =>
    Array.from({ length: BAR_COUNT }, (_, i) =>
      6 + Math.abs(Math.sin(i * 0.4) * Math.cos(i * 0.17)) * 54,
    ),
  );

  // Typewriter reveal — advance ~2 chars per 28ms. A 150-char line
  // resolves in ~2s, reasonably close to real speech cadence.
  const [revealed, setRevealed] = useState("");
  useEffect(() => {
    setRevealed("");
    const total = payload.script.length;
    let i = 0;
    const id = window.setInterval(() => {
      i += 2;
      if (i >= total) {
        setRevealed(payload.script);
        window.clearInterval(id);
        return;
      }
      setRevealed(payload.script.slice(0, i));
    }, 28);
    return () => window.clearInterval(id);
  }, [payload.script]);

  // ── Audio graph refs. Kept in refs because they belong to a single
  // playback; we tear them down and rebuild per play() call. ──────
  const ctxRef = useRef<AudioContext | null>(null);
  const srcRef = useRef<AudioBufferSourceNode | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const rafRef = useRef<number | null>(null);
  const endedRef = useRef(false);

  const tearDown = useCallback(() => {
    if (rafRef.current) cancelAnimationFrame(rafRef.current);
    rafRef.current = null;
    try {
      srcRef.current?.stop();
    } catch {
      // already stopped
    }
    try {
      srcRef.current?.disconnect();
      analyserRef.current?.disconnect();
    } catch {
      // no-op
    }
    srcRef.current = null;
    analyserRef.current = null;
    try {
      void ctxRef.current?.close();
    } catch {
      // no-op
    }
    ctxRef.current = null;
  }, []);

  const play = useCallback(async () => {
    // Fresh graph every time play() is invoked.
    tearDown();
    endedRef.current = false;
    setErr(null);
    setState("loading");

    const Ctor: typeof AudioContext | undefined =
      window.AudioContext ??
      (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!Ctor) {
      setErr("Web Audio API unavailable in this browser.");
      setState("error");
      return;
    }

    const ctx = new Ctor();
    ctxRef.current = ctx;

    // Unlock if the context started suspended. Must be called from
    // within the user-gesture promise chain, which play() always is.
    if (ctx.state === "suspended") {
      try {
        await ctx.resume();
      } catch {
        // Most browsers will let decodeAudioData proceed anyway;
        // push on.
      }
    }

    // Fetch + decode. This accepts data:, blob:, and http(s): URLs.
    let audioBuf: AudioBuffer;
    try {
      const res = await fetch(payload.audioUrl);
      const arr = await res.arrayBuffer();
      audioBuf = await ctx.decodeAudioData(arr);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setState("error");
      return;
    }

    const analyser = ctx.createAnalyser();
    analyser.fftSize = 256;
    // Tightened TS signature requires an ArrayBuffer, not ArrayBufferLike.
    const freq = new Uint8Array(new ArrayBuffer(analyser.frequencyBinCount));

    const source = ctx.createBufferSource();
    source.buffer = audioBuf;
    source.connect(analyser);
    analyser.connect(ctx.destination);

    source.onended = () => {
      if (endedRef.current) return;
      endedRef.current = true;
      setState("ended");
      // Small settle so the last typewriter chars aren't chopped.
      window.setTimeout(onComplete, 650);
    };

    srcRef.current = source;
    analyserRef.current = analyser;

    try {
      source.start();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setState("error");
      return;
    }

    // Final safety: if the autoplay policy is still blocking despite
    // resume(), the context will silently produce no output. We
    // detect this by re-checking state right after start().
    if (ctx.state !== "running") {
      tearDown();
      setState("awaiting-gesture");
      return;
    }

    setState("playing");

    const step = () => {
      if (!analyserRef.current) return;
      analyserRef.current.getByteFrequencyData(freq);
      const next: number[] = new Array(BAR_COUNT);
      const slice = freq.length / BAR_COUNT;
      for (let i = 0; i < BAR_COUNT; i++) {
        let sum = 0;
        let count = 0;
        const start = Math.floor(i * slice);
        const end = Math.max(start + 1, Math.floor((i + 1) * slice));
        for (let j = start; j < end && j < freq.length; j++) {
          sum += freq[j] ?? 0;
          count++;
        }
        const mean = count > 0 ? sum / count : 0;
        next[i] = 6 + (mean / 255) * 54;
      }
      setBars(next);
      rafRef.current = requestAnimationFrame(step);
    };
    rafRef.current = requestAnimationFrame(step);
  }, [payload.audioUrl, onComplete, tearDown]);

  // Attempt autoplay on mount / whenever a new cue arrives. If the
  // browser blocks it (common — our cues arrive seconds after the
  // originating gesture), fall back to the press-to-begin button.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await play();
      } catch (e) {
        if (cancelled) return;
        // eslint-disable-next-line no-console
        console.warn("narrator: autoplay blocked", e);
        setState("awaiting-gesture");
      }
    })();
    return () => {
      cancelled = true;
      tearDown();
    };
  }, [payload.cueId, play, tearDown]);

  const accent = useMemo(() => splitAccent(revealed), [revealed]);

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        background: "rgba(6, 8, 10, 0.72)",
        display: "flex",
        flexDirection: "column",
        alignItems: "stretch",
        justifyContent: "center",
        padding: "56px 120px",
        zIndex: 40,
        animation: "paperSlide 0.35s ease-out both",
      }}
    >
      <div className="t-eyebrow" style={{ color: rpColors.cyan, fontSize: 12, marginBottom: 20 }}>
        ▸ AUDIO TRANSMISSION · SYNTHESIZED VOICE · {payload.cue.toUpperCase()}
      </div>

      {/* Waveform */}
      <div style={{ display: "flex", alignItems: "center", gap: 5, height: 72, marginBottom: 48 }}>
        {bars.map((h, i) => (
          <div
            key={i}
            style={{
              width: 6,
              height: h,
              background: i < CYAN_BARS ? rpColors.cyan : rpColors.broadcastRule,
              opacity: i < CYAN_BARS ? 1 : 0.4,
              transition: "height 60ms linear",
              willChange: "height",
            }}
          />
        ))}
      </div>

      {/* Script */}
      <div
        style={{
          fontFamily: "var(--font-typewriter)",
          fontSize: 48,
          lineHeight: 1.35,
          color: rpColors.paper3,
          letterSpacing: "-0.005em",
          maxWidth: 1400,
        }}
      >
        &ldquo;{accent.head}
        {accent.accent && (
          <span
            style={{
              color: rpColors.stampRed,
              textDecoration: `underline ${rpColors.stampRed} 2px`,
              textUnderlineOffset: 8,
            }}
          >
            {accent.accent}
          </span>
        )}
        {revealed.length < payload.script.length && (
          <span
            className="animate-blink"
            style={{
              display: "inline-block",
              width: "0.55em",
              height: "1.1em",
              background: rpColors.paper3,
              verticalAlign: "-0.15em",
              marginLeft: 4,
            }}
          />
        )}
        &rdquo;
      </div>

      {/* Gesture prompt — shown when autoplay was refused. */}
      {state === "awaiting-gesture" && (
        <div
          style={{
            marginTop: 36,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            gap: 14,
          }}
        >
          <button
            onClick={() => {
              void play();
            }}
            style={{
              background: rpColors.cyan,
              color: rpColors.broadcast,
              border: `3px solid ${rpColors.cyan}`,
              padding: "16px 36px",
              fontFamily: "var(--font-display)",
              fontSize: 18,
              fontWeight: 700,
              letterSpacing: "0.18em",
              cursor: "pointer",
              boxShadow: "4px 4px 0 rgba(0,0,0,0.5)",
            }}
          >
            ▸ BEGIN TRANSMISSION
          </button>
          <div
            className="t-eyebrow"
            style={{ color: rpColors.inkFaded, fontSize: 11, letterSpacing: 2 }}
          >
            ONE PRESS REQUIRED · AUDIO POLICY
          </div>
        </div>
      )}

      {/* Error readout */}
      {state === "error" && err && (
        <div
          style={{
            marginTop: 28,
            padding: "10px 16px",
            border: `1px solid ${rpColors.stampRed}`,
            color: rpColors.stampRed,
            fontFamily: "var(--font-mono)",
            fontSize: 12,
            maxWidth: 680,
          }}
        >
          narrator playback failed · {err}
        </div>
      )}

      {/* Transport */}
      <div
        style={{
          position: "absolute",
          bottom: 40,
          left: 120,
          right: 120,
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-end",
        }}
      >
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 12,
            color: rpColors.inkFaded,
            lineHeight: 1.6,
          }}
        >
          <div style={{ color: rpColors.cyan }}>
            {state === "playing"
              ? "▸ SPEAKING"
              : state === "loading"
              ? "▸ SYNTHESIZING"
              : state === "awaiting-gesture"
              ? "▸ AWAITING OPERATOR"
              : state === "error"
              ? "▸ FAULT"
              : "▸ STAND BY"}
          </div>
          <div>CUE · {payload.cueId}</div>
          <div>MODEL · ORACLE-III</div>
        </div>
        <div style={{ display: "flex", gap: 12 }}>
          <button
            onClick={onSkip ?? onComplete}
            style={{
              background: "transparent",
              border: `1px solid ${rpColors.broadcastRule}`,
              color: rpColors.inkFaded,
              padding: "8px 14px",
              fontFamily: "var(--font-mono)",
              fontSize: 11,
              cursor: "pointer",
              letterSpacing: 1.2,
            }}
          >
            ⏭ CONTINUE
          </button>
        </div>
      </div>
    </div>
  );
}

// splitAccent finds the last standalone capitalised word of 4+ chars
// and paints it stamp-red with an underline — matches the handoff's
// typographic treatment. Only triggers once the sentence is "done"
// (ends in . or ,), so we don't accent a word mid-type.
function splitAccent(s: string): { head: string; accent: string | null } {
  const trimmed = s.trimEnd();
  if (!trimmed.endsWith(".") && !trimmed.endsWith(",")) {
    return { head: s, accent: null };
  }
  const match = trimmed.match(/\b([A-Z]{4,})(?=[.,]?$)/);
  if (!match) return { head: s, accent: null };
  const word = match[1]!;
  const head = s.slice(0, s.lastIndexOf(word));
  return { head, accent: s.slice(s.lastIndexOf(word)) };
}
