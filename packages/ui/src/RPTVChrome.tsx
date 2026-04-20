// RPTVChrome — the broadcast frame around every host-display screen.
// Header bar (REC blink + title + node/phase meta), scrollable body,
// footer ticker, and four corner registration marks. The whole thing
// sits on top of `.broadcast-tex .scanlines` so even empty content
// looks like it's coming through a CRT.

import type { CSSProperties, ReactNode } from "react";
import { rpColors } from "@replicant/tokens";
import { RPTicker } from "./RPTicker";

export interface RPTVChromeProps {
  children?: ReactNode;
  title?: string;
  nodeId?: string;
  phase?: string;
  /** Override the footer ticker items. */
  tickerItems?: string[];
  style?: CSSProperties;
}

const defaultTicker = [
  "DELEGATES REMINDED TO COOPERATE",
  "POLICIES VOTED IN OPEN SESSION",
  "ACCORD R-07 ACTIVE",
  "REPORT SYNTHETIC ANOMALIES",
  "THE COMMITTEE APPRECIATES YOUR COMPLIANCE",
];

export function RPTVChrome({
  children,
  title = "COMMITTEE BROADCAST",
  nodeId = "NODE 04-7",
  phase = "DAY 02",
  tickerItems = defaultTicker,
  style,
}: RPTVChromeProps) {
  return (
    <div
      className="broadcast-tex"
      style={{
        width: "100%",
        height: "100%",
        color: rpColors.paper3,
        display: "flex",
        flexDirection: "column",
        fontFamily: "var(--font-sans)",
        position: "relative",
        overflow: "hidden",
        ...style,
      }}
    >
      {/* header */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          padding: "14px 28px",
          borderBottom: `1px solid ${rpColors.broadcastRule}`,
          flexShrink: 0,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
          <span
            className="animate-blink"
            style={{ width: 10, height: 10, background: rpColors.stampRed, display: "inline-block" }}
          />
          <span className="t-eyebrow" style={{ color: rpColors.stampRed, fontSize: 11 }}>
            ● REC · LIVE TRANSMISSION
          </span>
        </div>
        <div className="t-eyebrow" style={{ color: rpColors.paper3, fontSize: 11 }}>
          {title}
        </div>
        <div
          style={{
            display: "flex",
            gap: 18,
            fontFamily: "var(--font-mono)",
            fontSize: 11,
            color: rpColors.inkFaded,
          }}
        >
          <span>{nodeId}</span>
          <span style={{ color: rpColors.cyan }}>{phase}</span>
        </div>
      </div>

      {/* body — scanline overlay scoped to THIS region only. The
          mix-blend-mode ::after must not extend over the footer
          ticker: blending against a moving element every frame
          forces a full re-composite per frame and makes the
          marquee shimmer. Keeping scanlines off the ticker keeps
          the ticker on its own clean compositor layer. */}
      <div
        className="scanlines"
        style={{ flex: 1, minHeight: 0, position: "relative" }}
      >
        {children}
      </div>

      {/* footer ticker — outside the scanlines on purpose */}
      <div style={{ borderTop: `1px solid ${rpColors.broadcastRule}`, flexShrink: 0 }}>
        <RPTicker bg={rpColors.broadcast2} fg={rpColors.cyan} items={tickerItems} />
      </div>

      {/* corner registration marks */}
      {(
        [
          ["tl", { top: 8, left: 8 }],
          ["tr", { top: 8, right: 8 }],
          ["bl", { bottom: 8, left: 8 }],
          ["br", { bottom: 8, right: 8 }],
        ] as const
      ).map(([key, pos]) => (
        <div
          key={key}
          aria-hidden="true"
          style={{
            position: "absolute",
            width: 16,
            height: 16,
            border: `1px solid ${rpColors.cyan}`,
            opacity: 0.5,
            ...pos,
          }}
        />
      ))}
    </div>
  );
}
