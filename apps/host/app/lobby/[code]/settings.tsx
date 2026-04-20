// Host settings panel — modal overlay for the lobby.
//
// Two concern groups:
//   1. Game rules (RulesConfig, lobby-only). PUT /host/rules on save.
//   2. Volume (client-side, localStorage-backed via lib/volume).
//
// Kept self-contained: the lobby page mounts <SettingsButton game=… />
// and we handle the open/close + form state in here. No external state
// library.

"use client";

import { useEffect, useState } from "react";
import type { Game, RulesConfig } from "@replicant/schema";
import { rpColors } from "@replicant/tokens";
import { HostApi, ApiError } from "@/lib/api";
import {
  DEFAULT_VOLUME,
  getVolume,
  setVolume,
  type VolumeSettings,
} from "@/lib/volume";

export function SettingsButton({
  game,
  token,
  onRulesUpdated,
  disabled,
}: {
  game: Game | null;
  token: string | null;
  onRulesUpdated?: (g: Game) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        disabled={disabled}
        onClick={() => setOpen(true)}
        aria-label="Open settings"
        style={{
          position: "absolute",
          top: 12,
          right: 12,
          zIndex: 10,
          background: "transparent",
          border: `1px solid ${rpColors.broadcastRule}`,
          color: rpColors.inkFaded,
          padding: "6px 12px",
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          letterSpacing: 1.4,
          cursor: disabled ? "not-allowed" : "pointer",
          opacity: disabled ? 0.4 : 1,
        }}
      >
        ⚙ SETTINGS
      </button>
      {open && (
        <SettingsModal
          game={game}
          token={token}
          onClose={() => setOpen(false)}
          onRulesUpdated={onRulesUpdated}
        />
      )}
    </>
  );
}

function SettingsModal({
  game,
  token,
  onClose,
  onRulesUpdated,
}: {
  game: Game | null;
  token: string | null;
  onClose: () => void;
  onRulesUpdated?: (g: Game) => void;
}) {
  // Rules form state. Lifted from game.rules on first render so the UI
  // can mutate locally without round-tripping per keystroke.
  const [draft, setDraft] = useState<RulesConfig | null>(
    game?.rules ? { ...game.rules } : null,
  );
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  // Volume slider state is a local mirror of the persisted settings.
  // We write through setVolume() on every change so audio elements
  // pick up the new values immediately via their subscribeVolume hook.
  const [volume, setVol] = useState<VolumeSettings>(() =>
    typeof window === "undefined" ? DEFAULT_VOLUME : getVolume(),
  );

  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  const editable = game?.status === "lobby";

  const saveRules = async () => {
    if (!game || !token || !draft) return;
    setSaving(true);
    setErr(null);
    try {
      const api = new HostApi({ token });
      const res = await api.updateRules(game.gameId, draft);
      onRulesUpdated?.(res.game);
    } catch (e) {
      setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(6, 8, 10, 0.78)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 100,
        padding: 20,
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        style={{
          width: "min(720px, 100%)",
          maxHeight: "90vh",
          overflow: "auto",
          background: rpColors.paper3,
          color: rpColors.ink,
          padding: "28px 32px",
          border: `2px solid ${rpColors.ink}`,
          boxShadow: "6px 6px 0 rgba(0,0,0,0.55)",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 18,
          }}
        >
          <div className="t-eyebrow" style={{ color: rpColors.stampRed }}>
            ◼ COMMITTEE SETTINGS · FORM R-99
          </div>
          <button
            onClick={onClose}
            aria-label="Close settings"
            style={{
              background: "transparent",
              border: "none",
              color: rpColors.ink,
              fontSize: 24,
              cursor: "pointer",
              lineHeight: 1,
            }}
          >
            ×
          </button>
        </div>

        <Section title="AUDIO">
          <VolumeSlider
            label="Master"
            value={volume.master}
            onChange={(v) => {
              const next = { ...volume, master: v };
              setVol(next);
              setVolume({ master: v });
            }}
          />
          <VolumeSlider
            label="Narrator"
            value={volume.narrator}
            onChange={(v) => {
              const next = { ...volume, narrator: v };
              setVol(next);
              setVolume({ narrator: v });
            }}
          />
          <VolumeSlider
            label="Background (music)"
            value={volume.music}
            hint="Reserved for the upcoming soundtrack."
            onChange={(v) => {
              const next = { ...volume, music: v };
              setVol(next);
              setVolume({ music: v });
            }}
          />
        </Section>

        <Section title="GAME RULES">
          {!editable && (
            <Note>
              Rules are locked once the game has started. Return to the
              lobby before a new session to edit.
            </Note>
          )}
          {editable && draft && (
            <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
              <ToggleRow
                label="Cable Phase (every round)"
                hint="Opens a 45s writing window between nomination and vote. LLM picks one cable to leak."
                checked={draft.cablePhaseMode === "every_round"}
                onChange={(on) =>
                  setDraft({ ...draft, cablePhaseMode: on ? "every_round" : "disabled" })
                }
              />
              <NumberRow
                label="Cable Phase duration (seconds)"
                value={draft.cablePhaseDurationSec ?? 45}
                min={15}
                max={120}
                onChange={(v) => setDraft({ ...draft, cablePhaseDurationSec: v })}
                disabled={draft.cablePhaseMode === "disabled"}
              />
              <NumberRow
                label="Silence chance (0–100%)"
                value={Math.round((draft.cableLeakSilenceChance ?? 0.15) * 100)}
                min={0}
                max={100}
                onChange={(v) =>
                  setDraft({ ...draft, cableLeakSilenceChance: v / 100 })
                }
                disabled={draft.cablePhaseMode === "disabled"}
              />
              <ToggleRow
                label="Anonymous leaks"
                hint="When on, the Committee broadcasts a leaked cable without naming its author. Off makes the leak a direct outing."
                checked={draft.anonymousCableLeaks !== false}
                onChange={(on) => setDraft({ ...draft, anonymousCableLeaks: on })}
              />
              <ToggleRow
                label="Disable AI narration"
                hint="When on, the Committee plays silently — no opening, closing, execution eulogy, or cable-leak voiceover. The board still shows text and stamps; only the synthesised voice is skipped."
                checked={!!draft.disableNarrator}
                onChange={(on) => setDraft({ ...draft, disableNarrator: on })}
              />
              <ToggleRow
                label="Enable Singularity (3-faction)"
                hint="6+ players. Adds one kingmaker role — wins alone if they take the Envoy's seat post-codes."
                checked={!!draft.enableSingularity}
                onChange={(on) => setDraft({ ...draft, enableSingularity: on })}
              />
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "1fr 1fr",
                  gap: 12,
                }}
              >
                <NumberRow
                  label="Human policies to win"
                  value={draft.humanPoliciesToWin}
                  min={1}
                  max={10}
                  onChange={(v) => setDraft({ ...draft, humanPoliciesToWin: v })}
                />
                <NumberRow
                  label="AI policies to win"
                  value={draft.aiPoliciesToWin}
                  min={1}
                  max={10}
                  onChange={(v) => setDraft({ ...draft, aiPoliciesToWin: v })}
                />
                <NumberRow
                  label="Codes transfer after … AI policies"
                  value={draft.codesTransferAt}
                  min={0}
                  max={draft.aiPoliciesToWin - 1}
                  onChange={(v) => setDraft({ ...draft, codesTransferAt: v })}
                />
                <NumberRow
                  label="Veto unlocks after … AI policies"
                  value={draft.vetoUnlockAt}
                  min={0}
                  max={draft.aiPoliciesToWin}
                  onChange={(v) => setDraft({ ...draft, vetoUnlockAt: v })}
                />
              </div>

              {err && (
                <div
                  style={{
                    border: `1px solid ${rpColors.stampRed}`,
                    background: "rgba(179,39,36,0.08)",
                    color: rpColors.stampRed,
                    padding: "8px 12px",
                    fontFamily: "var(--font-mono)",
                    fontSize: 12,
                  }}
                >
                  {err}
                </div>
              )}

              <div
                style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 6 }}
              >
                <button
                  onClick={() => game?.rules && setDraft({ ...game.rules })}
                  disabled={saving}
                  style={btnStyle(rpColors.inkFaded)}
                >
                  Revert
                </button>
                <button
                  onClick={saveRules}
                  disabled={saving}
                  style={btnStyle(rpColors.stampBlue)}
                >
                  {saving ? "Saving…" : "Save rules"}
                </button>
              </div>
            </div>
          )}
        </Section>
      </div>
    </div>
  );
}

// ─── bits ────────────────────────────────────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div
      style={{
        borderTop: `1px solid ${rpColors.paperLine}`,
        paddingTop: 16,
        marginTop: 10,
      }}
    >
      <div
        className="t-eyebrow"
        style={{ color: rpColors.inkFaded, fontSize: 10, marginBottom: 10 }}
      >
        {title}
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>{children}</div>
    </div>
  );
}

function VolumeSlider({
  label,
  value,
  hint,
  onChange,
}: {
  label: string;
  value: number;
  hint?: string;
  onChange: (v: number) => void;
}) {
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
        <span style={{ fontFamily: "var(--font-sans)", fontSize: 14, color: rpColors.ink }}>
          {label}
        </span>
        <span
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 12,
            color: rpColors.inkFaded,
          }}
        >
          {Math.round(value * 100)}%
        </span>
      </div>
      <input
        type="range"
        min={0}
        max={100}
        value={Math.round(value * 100)}
        onChange={(e) => onChange(Number(e.target.value) / 100)}
        style={{ accentColor: rpColors.stampBlue }}
      />
      {hint && (
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            color: rpColors.inkFaded,
          }}
        >
          {hint}
        </div>
      )}
    </label>
  );
}

function ToggleRow({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (on: boolean) => void;
}) {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 12,
        cursor: "pointer",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        style={{ marginTop: 3 }}
      />
      <div>
        <div style={{ fontFamily: "var(--font-sans)", fontSize: 14, color: rpColors.ink }}>
          {label}
        </div>
        {hint && (
          <div
            style={{
              fontFamily: "var(--font-mono)",
              fontSize: 10,
              color: rpColors.inkFaded,
              marginTop: 2,
            }}
          >
            {hint}
          </div>
        )}
      </div>
    </label>
  );
}

function NumberRow({
  label,
  value,
  min,
  max,
  onChange,
  disabled,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
  disabled?: boolean;
}) {
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 4, opacity: disabled ? 0.45 : 1 }}>
      <span style={{ fontFamily: "var(--font-sans)", fontSize: 13, color: rpColors.ink }}>
        {label}
      </span>
      <input
        type="number"
        min={min}
        max={max}
        value={value}
        disabled={disabled}
        onChange={(e) => {
          const raw = Number(e.target.value);
          if (!Number.isFinite(raw)) return;
          onChange(Math.max(min, Math.min(max, Math.round(raw))));
        }}
        style={{
          padding: "6px 8px",
          border: `1px solid ${rpColors.ink}`,
          background: rpColors.paper2,
          fontFamily: "var(--font-mono)",
          fontSize: 14,
          color: rpColors.ink,
        }}
      />
    </label>
  );
}

function Note({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        border: `1px dashed ${rpColors.inkFaded}`,
        padding: "8px 12px",
        fontFamily: "var(--font-mono)",
        fontSize: 11,
        color: rpColors.inkFaded,
        lineHeight: 1.5,
      }}
    >
      {children}
    </div>
  );
}

function btnStyle(accent: string): React.CSSProperties {
  return {
    background: accent,
    color: rpColors.paper3,
    border: `2px solid ${accent}`,
    padding: "8px 14px",
    fontFamily: "var(--font-display)",
    fontWeight: 700,
    fontSize: 13,
    letterSpacing: "0.12em",
    textTransform: "uppercase",
    cursor: "pointer",
  };
}
