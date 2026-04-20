// panels.tsx — phase-specific action panels used by the mobile game
// screen. Each panel is a small, focused component that knows how to
// render its phase and call one of the MobileApi action methods.
//
// The patterns that repeat here (title + memo copy + action area) are
// extracted into the `Panel` shell at the bottom of this file.

"use client";

import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { rpColors } from "@replicant/tokens";
import { RPButton, RPCheckbox, RPStamp } from "@replicant/ui";
import type {
  ExecutiveActionType,
  Game,
  Party,
  Player,
  PolicyType,
  Role,
  WinCondition,
} from "@replicant/schema";

// ─── helpers ───────────────────────────────────────────────────────

function policyColor(p: PolicyType): string {
  return p === "human" ? rpColors.stampBlue : rpColors.stampRed;
}

function policyLabel(p: PolicyType): string {
  return p === "human" ? "HUMAN" : "AI";
}

export interface Candidate {
  player: Player;
  disabled?: boolean;
  /** Note shown next to the name when disabled, e.g. "term-limited" */
  note?: string;
}

// ─── shared Panel shell ────────────────────────────────────────────

/**
 * Standard mobile action panel — eyebrow + large display title + a
 * memo-typed sub-line + body content + action area. Every phase
 * screen is composed of this.
 */
export function Panel({
  eyebrow,
  eyebrowColor = rpColors.stampRed,
  title,
  memo,
  children,
  action,
}: {
  eyebrow: string;
  eyebrowColor?: string;
  title: string;
  memo?: ReactNode;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <main
      className="animate-paper-slide"
      style={{
        padding: "22px 22px 28px",
        display: "flex",
        flexDirection: "column",
        gap: 14,
        flex: 1,
        minHeight: 0,
      }}
    >
      <div className="t-eyebrow" style={{ color: eyebrowColor, fontSize: 10 }}>
        ◼ {eyebrow}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 36,
          color: rpColors.ink,
          lineHeight: 0.95,
        }}
      >
        {title.split("\n").map((l, i) => (
          <div key={i}>{l}</div>
        ))}
      </div>
      {memo && (
        <div
          className="t-memo"
          style={{ fontSize: 14, color: rpColors.inkSoft, lineHeight: 1.55 }}
        >
          {memo}
        </div>
      )}
      {children && (
        <div style={{ display: "flex", flexDirection: "column", gap: 10, marginTop: 4 }}>
          {children}
        </div>
      )}
      {action && <div style={{ marginTop: "auto" }}>{action}</div>}
    </main>
  );
}

// ─── "waiting" placeholder ─────────────────────────────────────────

export function WaitFor({ label, sub }: { label: string; sub?: string }) {
  return (
    <Panel
      eyebrow="STANDBY"
      eyebrowColor={rpColors.inkFaded}
      title={label}
      memo={sub ?? '"The Department will notify you when your turn arrives."'}
    />
  );
}

// ─── Cable Phase compose ─────────────────────────────────────────

/**
 * CablePanel — the mobile view for Cable Phase. Shows a live
 * countdown, a text input, and a submit button. Each submission is
 * fire-and-forget: the server whispers back an Ack=true payload that
 * flips the "queued" row on this device; the player may submit
 * multiple cables during the window.
 *
 * Not wired to faction/role yet — step 3d will inject an LLM-generated
 * cover prompt into the placeholder for humans and an open cabal
 * input for Replicants.
 */
export function CablePanel({
  deadline,
  sent,
  onSubmit,
  disabled,
}: {
  deadline: string | null;
  /** Cables this device has successfully submitted this phase. */
  sent: Array<{ messageId: string; body: string }>;
  onSubmit: (body: string) => Promise<void>;
  disabled?: boolean;
}) {
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [remaining, setRemaining] = useState<string>("--:--");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Tick the countdown once a second off the server-provided deadline.
  useEffect(() => {
    if (!deadline) {
      setRemaining("--:--");
      return;
    }
    const tick = () => {
      const remainMs = Math.max(0, Date.parse(deadline) - Date.now());
      const s = Math.floor(remainMs / 1000);
      setRemaining(`${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`);
    };
    tick();
    const id = setInterval(tick, 500);
    return () => clearInterval(id);
  }, [deadline]);

  const submit = async () => {
    const body = draft.trim();
    if (!body || busy || disabled) return;
    setBusy(true);
    try {
      await onSubmit(body);
      setDraft("");
      textareaRef.current?.focus();
    } finally {
      setBusy(false);
    }
  };

  return (
    <main
      className="animate-paper-slide"
      style={{
        padding: "24px 22px 32px",
        display: "flex",
        flexDirection: "column",
        gap: 14,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end" }}>
        <div className="t-eyebrow" style={{ color: rpColors.stampRed }}>
          ▼ CABLE PHASE · TRANSMIT
        </div>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontWeight: 700,
            fontSize: 20,
            color: rpColors.ink,
            letterSpacing: "0.06em",
          }}
        >
          {remaining}
        </div>
      </div>

      <div
        className="t-memo"
        style={{ fontSize: 13, color: rpColors.inkSoft, lineHeight: 1.45, maxWidth: 360 }}
      >
        &quot;Compose any cable you wish the Department to review. One may be
        broadcast. You may submit more than once.&quot;
      </div>

      <textarea
        ref={textareaRef}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        placeholder="Issue a statement…"
        rows={4}
        disabled={busy || disabled}
        style={{
          width: "100%",
          resize: "vertical",
          minHeight: 96,
          padding: 12,
          border: `2px solid ${rpColors.ink}`,
          background: rpColors.paper3,
          fontFamily: "var(--font-typewriter)",
          fontSize: 16,
          color: rpColors.ink,
          lineHeight: 1.5,
          outline: "none",
          boxShadow: "3px 3px 0 rgba(0,0,0,0.08)",
        }}
      />

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            color: rpColors.inkFaded,
            letterSpacing: 1.3,
          }}
        >
          QUEUED · {String(sent.length).padStart(2, "0")}
        </div>
        <RPButton onClick={submit} disabled={busy || disabled || !draft.trim()}>
          {busy ? "…TRANSMITTING" : "▸ TRANSMIT"}
        </RPButton>
      </div>

      {sent.length > 0 && (
        <div
          style={{
            borderTop: `1px dashed ${rpColors.paperLine}`,
            paddingTop: 10,
            display: "flex",
            flexDirection: "column",
            gap: 6,
          }}
        >
          <div
            className="t-eyebrow"
            style={{ color: rpColors.inkFaded, fontSize: 10, marginBottom: 4 }}
          >
            PRIOR TRANSMISSIONS
          </div>
          {sent.slice(-4).map((m) => (
            <div
              key={m.messageId}
              style={{
                fontFamily: "var(--font-typewriter)",
                fontSize: 13,
                color: rpColors.inkSoft,
                lineHeight: 1.35,
                paddingLeft: 10,
                borderLeft: `2px solid ${rpColors.paperLine}`,
              }}
            >
              {m.body}
            </div>
          ))}
        </div>
      )}
    </main>
  );
}

// ─── nominate + executive target + vote-for-player pickers ────────

/**
 * A candidate row with an RPCheckbox, portrait, subject number, and
 * display name. Used by Nominate + Executive-target pickers.
 */
export function CandidateRow({
  candidate,
  selected,
  onSelect,
  suffix,
}: {
  candidate: Candidate;
  selected: boolean;
  onSelect: () => void;
  suffix?: ReactNode;
}) {
  const p = candidate.player;
  const disabled = !!candidate.disabled;
  return (
    <button
      onClick={disabled ? undefined : onSelect}
      disabled={disabled}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        background: selected ? rpColors.paper2 : rpColors.paper3,
        border: `2px solid ${selected ? rpColors.stampRed : rpColors.ink}`,
        borderLeft: selected ? `10px solid ${rpColors.stampRed}` : `2px solid ${rpColors.ink}`,
        padding: selected ? "10px 12px 10px 8px" : "10px 12px",
        position: "relative",
        opacity: disabled ? 0.55 : 1,
        textAlign: "left",
        width: "100%",
        cursor: disabled ? "not-allowed" : "pointer",
        font: "inherit",
        color: "inherit",
      }}
    >
      <RPCheckbox checked={selected} />
      <div
        style={{
          width: 42,
          height: 42,
          background: rpColors.ink,
          color: rpColors.paper3,
          border: `1.5px solid ${rpColors.ink}`,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 20,
        }}
      >
        {p.displayName.trim().slice(0, 1).toUpperCase() || "?"}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 9,
            color: rpColors.inkFaded,
            letterSpacing: 1.2,
          }}
        >
          SUBJECT #{String(p.seat + 1).padStart(2, "0")}
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 600,
            fontSize: 17,
            color: rpColors.ink,
            letterSpacing: "0.02em",
          }}
        >
          {p.displayName.toUpperCase()}
          {candidate.note && (
            <span style={{ fontSize: 10, color: rpColors.inkFaded, letterSpacing: 1, marginLeft: 6 }}>
              · {candidate.note}
            </span>
          )}
        </div>
      </div>
      {suffix}
    </button>
  );
}

// ─── policy picker (for legislative phases) ───────────────────────

export function PolicyButton({
  policy,
  label,
  onClick,
  hint,
}: {
  policy: PolicyType;
  label: string;
  onClick: () => void;
  hint?: string;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 12,
        background: rpColors.paper3,
        border: `2px solid ${policyColor(policy)}`,
        borderLeft: `10px solid ${policyColor(policy)}`,
        padding: "14px 16px 14px 12px",
        font: "inherit",
        color: "inherit",
        cursor: "pointer",
        width: "100%",
        textAlign: "left",
      }}
    >
      <div>
        <div
          className="t-eyebrow"
          style={{ color: policyColor(policy), fontSize: 10, marginBottom: 4 }}
        >
          POLICY · {policyLabel(policy)}
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 22,
            color: rpColors.ink,
            letterSpacing: "0.02em",
          }}
        >
          {label}
        </div>
        {hint && (
          <div
            className="t-memo"
            style={{ fontSize: 12, color: rpColors.inkFaded, marginTop: 4 }}
          >
            {hint}
          </div>
        )}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 32,
          color: policyColor(policy),
        }}
      >
        ▸
      </div>
    </button>
  );
}

// ─── ballot · yea / nay ───────────────────────────────────────────
//
// The component's `onYea`/`onNay` callbacks still map to the engine's
// "ja" / "nein" VoteChoice wire values — we kept the backend enum
// intact to avoid data migration. Only the displayed labels reflect
// the Earth Policy Committee theme.

export function JaNein({
  onYea,
  onNay,
  yeaLabel = "YEA",
  nayLabel = "NAY",
}: {
  onYea: () => void;
  onNay: () => void;
  yeaLabel?: string;
  nayLabel?: string;
}) {
  const btnStyle = (accent: string): CSSProperties => ({
    flex: 1,
    padding: "22px 16px",
    background: rpColors.paper3,
    border: `3px solid ${accent}`,
    color: accent,
    fontFamily: "var(--font-display)",
    fontWeight: 700,
    fontSize: 28,
    letterSpacing: "0.1em",
    textTransform: "uppercase",
    cursor: "pointer",
    boxShadow: "3px 3px 0 rgba(28,26,21,0.25)",
  });
  return (
    <div style={{ display: "flex", gap: 12 }}>
      <button onClick={onYea} style={btnStyle(rpColors.stampBlue)}>
        ✓ {yeaLabel}
      </button>
      <button onClick={onNay} style={btnStyle(rpColors.stampRed)}>
        ✗ {nayLabel}
      </button>
    </div>
  );
}

// ─── terminated / spectator ───────────────────────────────────────

/**
 * Terminated — eliminated player's spectator view. Receives the same
 * SSE stream as living players, so round/policy counts and the current
 * President name update live. Kept read-only: no action controls,
 * just a silent feed.
 */
export function Terminated({
  game,
  presidentName,
  chancellorName,
  aliveCount,
  totalCount,
}: {
  game: Game | null;
  presidentName: string;
  chancellorName: string;
  aliveCount: number;
  totalCount: number;
}) {
  const phaseLabel = game ? terminatedPhaseLabel(game.phase) : "AWAITING";
  return (
    <main
      className="animate-paper-slide"
      style={{
        padding: "32px 22px 96px",
        display: "flex",
        flexDirection: "column",
        gap: 18,
        alignItems: "center",
      }}
    >
      <RPStamp variant="red" rotate={-8} size={28} animate>
        TERMINATED
      </RPStamp>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 40,
          color: rpColors.ink,
          textAlign: "center",
          lineHeight: 0.95,
          letterSpacing: "-0.01em",
        }}
      >
        PROCESSING
        <br />
        COMPLETE
      </div>
      <div
        className="t-memo"
        style={{
          fontSize: 14,
          color: rpColors.inkSoft,
          lineHeight: 1.5,
          textAlign: "center",
          maxWidth: 320,
        }}
      >
        &quot;Your ballot is void. You may observe the remainder of the
        assembly&apos;s deliberations; you may not speak of them afterward.&quot;
      </div>

      {/* Live broadcast slab — mimics the manila debrief form. */}
      <div
        style={{
          marginTop: 8,
          alignSelf: "stretch",
          border: `1px solid ${rpColors.ink}`,
          background: rpColors.paper3,
          boxShadow: "3px 3px 0 rgba(0,0,0,0.08)",
        }}
      >
        <div
          style={{
            padding: "10px 14px",
            borderBottom: `1px solid ${rpColors.ink}`,
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            color: rpColors.inkFaded,
            letterSpacing: 1.3,
          }}
        >
          <span>◼ SPECTATOR LOG · RO</span>
          <span>{phaseLabel}</span>
        </div>
        <div
          style={{
            padding: 16,
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: 12,
          }}
        >
          <SpecStat label="ROUND" value={game ? `D${String(game.round).padStart(2, "0")}` : "—"} />
          <SpecStat
            label="ACTIVE"
            value={`${String(aliveCount).padStart(2, "0")}/${String(totalCount).padStart(2, "0")}`}
          />
          <SpecStat
            label="HUMAN PROTOCOLS"
            value={`${String(game?.humanPoliciesEnacted ?? 0).padStart(2, "0")}/05`}
            color={rpColors.stampBlue}
          />
          <SpecStat
            label="AI PROTOCOLS"
            value={`${String(game?.aiPoliciesEnacted ?? 0).padStart(2, "0")}/06`}
            color={rpColors.stampRed}
          />
          <SpecStat label="PRESIDENT" value={presidentName.toUpperCase() || "—"} span={2} />
          {chancellorName && (
            <SpecStat label="CHANCELLOR" value={chancellorName.toUpperCase()} span={2} />
          )}
        </div>
      </div>
    </main>
  );
}

function SpecStat({
  label,
  value,
  color = rpColors.ink,
  span = 1,
}: {
  label: string;
  value: string;
  color?: string;
  span?: 1 | 2;
}) {
  return (
    <div
      style={{
        gridColumn: span === 2 ? "span 2" : undefined,
        borderLeft: `2px solid ${color}`,
        paddingLeft: 10,
        minWidth: 0,
      }}
    >
      <div
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: 9,
          color: rpColors.inkFaded,
          letterSpacing: 1.4,
          marginBottom: 2,
        }}
      >
        {label}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 20,
          color,
          lineHeight: 1,
          letterSpacing: "0.02em",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
      >
        {value}
      </div>
    </div>
  );
}

function terminatedPhaseLabel(phase: Game["phase"]): string {
  switch (phase) {
    case "lobby":
      return "INTAKE";
    case "nomination":
      return "NOMINATION";
    case "cable_phase":
      return "CABLES";
    case "election":
      return "TRIBUNAL";
    case "legislative_president":
      return "SORTING ROOM";
    case "legislative_chancellor":
      return "DRAFTING FLOOR";
    case "veto_requested":
      return "VETO";
    case "executive_action":
      return "EXEC ORDER";
    case "game_over":
      return "FINAL";
  }
}

// ─── game over banner ─────────────────────────────────────────────

export function GameOverPanel({
  winner,
  condition,
  myParty,
}: {
  winner: Party | null;
  condition: WinCondition | null;
  myParty: Party | null;
}) {
  const victorious = myParty && winner && myParty === winner;
  const humansWin = winner === "human";
  const accent = humansWin ? rpColors.stampBlue : rpColors.stampRed;
  return (
    <main
      className="animate-paper-slide"
      style={{ padding: "36px 22px", display: "flex", flexDirection: "column", gap: 16, alignItems: "center" }}
    >
      <div className="t-eyebrow" style={{ color: accent }}>
        ◼ FINAL DISPOSITION
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 52,
          color: accent,
          letterSpacing: "-0.01em",
          lineHeight: 0.9,
          textAlign: "center",
        }}
      >
        {humansWin ? "HUMANS" : "REPLICANTS"}
        <br />
        PREVAIL
      </div>
      <div
        className="t-memo"
        style={{ fontSize: 14, color: rpColors.inkSoft, lineHeight: 1.5, textAlign: "center", maxWidth: 320 }}
      >
        {condition ? winMemo(condition) : "The Department has filed its findings."}
      </div>
      <div
        style={{
          marginTop: 8,
          padding: "12px 20px",
          border: `2px solid ${accent}`,
          color: accent,
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 16,
          letterSpacing: "0.12em",
          textTransform: "uppercase",
        }}
      >
        {victorious ? "▸ Your file reflects well" : "◻ Your file is noted"}
      </div>
    </main>
  );
}

function winMemo(c: WinCondition): string {
  switch (c) {
    case "human_policies":
      return '"Five Human protocols ratified. The synthetic threat has been contained."';
    case "ai_policies":
      return '"Six AI protocols passed. The transition has been completed in good order."';
    case "rogue_elected_chancellor":
      return '"The Prime has been elected to the Chancellery. The Department yields its keys."';
    case "rogue_executed":
      return '"The Prime has been processed. The population is returned to its former allocation."';
    case "singularity_kingmaker":
      return '"The Singularity has taken the Chancellery. Neither faction prevails — a solitary victor claims the codes."';
  }
}

// ─── role reminder bar ────────────────────────────────────────────

export function RoleReminder({
  role,
  party,
  onHold,
  onRelease,
  peeking,
}: {
  role: Role;
  party: Party | null;
  onHold: () => void;
  onRelease: () => void;
  peeking: boolean;
}) {
  const accent = accentFor(role);
  return (
    <div
      style={{
        position: "sticky",
        bottom: 0,
        background: rpColors.paper2,
        borderTop: `1px dashed ${rpColors.paperLine}`,
        padding: "6px 16px",
        display: "flex",
        alignItems: "center",
        gap: 10,
        fontFamily: "var(--font-mono)",
        fontSize: 10,
        color: rpColors.inkFaded,
        letterSpacing: 1.4,
        zIndex: 5,
      }}
    >
      <span>PASSPORT:</span>
      <span
        onPointerDown={(e) => {
          (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
          onHold();
        }}
        onPointerUp={onRelease}
        onPointerCancel={onRelease}
        onPointerLeave={onRelease}
        style={{
          cursor: "pointer",
          userSelect: "none",
          touchAction: "none",
          display: "inline-flex",
          alignItems: "center",
          gap: 8,
          padding: "4px 10px",
          background: peeking ? accent : rpColors.ink,
          color: peeking ? rpColors.paper3 : rpColors.paper3,
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          letterSpacing: "0.14em",
          fontSize: 12,
          WebkitTapHighlightColor: "transparent",
        }}
      >
        {peeking ? (
          <>
            {labelFor(role)} · {party?.toUpperCase() ?? ""}
          </>
        ) : (
          <>◼ HOLD TO REVEAL</>
        )}
      </span>
    </div>
  );
}

function labelFor(r: Role): string {
  switch (r) {
    case "human":
      return "HUMAN";
    case "ai":
      return "REPLICANT";
    case "rogue":
      return "PRIME";
    case "singularity":
      return "SINGULARITY";
  }
}

function accentFor(r: Role): string {
  switch (r) {
    case "human":
      return rpColors.stampBlue;
    case "ai":
      return rpColors.stampRed;
    case "rogue":
    case "singularity":
      return rpColors.amber;
  }
}

// ─── executive-action copy ────────────────────────────────────────

export const EXEC_COPY: Record<ExecutiveActionType, { title: string; memo: string }> = {
  investigate_loyalty: {
    title: "INVESTIGATE\nLOYALTY",
    memo: '"Select one citizen for file inspection. Their party affiliation will be disclosed to you alone."',
  },
  special_election: {
    title: "SPECIAL\nELECTION",
    memo: '"Appoint the next Presidential candidate. The Department will restore normal rotation afterward."',
  },
  policy_peek: {
    title: "POLICY\nPEEK",
    memo: '"You may review the next three policies to be drawn. Disclosure to fellow citizens is inadvisable."',
  },
  execution: {
    title: "EXECUTION",
    memo: '"Nominate one citizen for immediate processing. The Department will action this without further review."',
  },
  top_deck: {
    title: "TOP DECK",
    memo: '"The Department will action the next policy automatically."',
  },
};

export { policyColor, policyLabel };
