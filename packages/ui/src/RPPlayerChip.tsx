// RPPlayerChip — a citizen ID card. Avatar (initial) + SUBJECT number
// + name + status line. The avatar uses the player's accent color with
// a crosshatch overlay (same as the design handoff).

import { rpColors } from "@replicant/tokens";

export type RPPlayerStatus = "ALIVE" | "TERMINATED" | "JOINING" | "ACTIVE";

export interface RPPlayerChipProps {
  name?: string;
  /** Subject number without the "#" prefix (we prepend it). */
  num?: string;
  status?: RPPlayerStatus;
  portrait?: string;
  accent?: string;
  small?: boolean;
}

export function RPPlayerChip({
  name = "SUBJECT 04",
  num = "#04",
  status = "ALIVE",
  portrait = "H",
  accent = rpColors.ink,
  small = false,
}: RPPlayerChipProps) {
  const h = small ? 56 : 78;
  const statusDot =
    status === "ALIVE" || status === "ACTIVE"
      ? rpColors.stampGreen
      : status === "TERMINATED"
      ? rpColors.stampRed
      : rpColors.inkFaded;

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        background: rpColors.paper3,
        border: `1.5px solid ${rpColors.ink}`,
        padding: small ? "6px 10px 6px 6px" : "8px 14px 8px 8px",
        boxShadow: "2px 2px 0 rgba(28,26,21,0.3)",
        minHeight: h,
        position: "relative",
      }}
    >
      <div
        style={{
          width: small ? 44 : 62,
          height: small ? 44 : 62,
          flexShrink: 0,
          background: accent,
          color: rpColors.paper3,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: small ? 20 : 28,
          border: `1.5px solid ${rpColors.ink}`,
          position: "relative",
          overflow: "hidden",
        }}
      >
        <span style={{ position: "relative", zIndex: 1 }}>{portrait}</span>
        <div
          style={{
            position: "absolute",
            inset: 0,
            background:
              "repeating-linear-gradient(45deg, transparent 0 3px, rgba(255,255,255,0.08) 3px 4px)",
          }}
        />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: small ? 9 : 10,
            color: rpColors.inkFaded,
            letterSpacing: 1.4,
          }}
        >
          SUBJECT {num}
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 600,
            fontSize: small ? 15 : 18,
            color: rpColors.ink,
            textTransform: "uppercase",
            letterSpacing: "0.02em",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          {name}
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 5, marginTop: 2 }}>
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: "50%",
              background: statusDot,
              display: "inline-block",
            }}
          />
          <span className="t-eyebrow" style={{ fontSize: 9, color: rpColors.inkFaded }}>
            {status}
          </span>
        </div>
      </div>
    </div>
  );
}
