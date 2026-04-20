// RPPaper — generic paper surface. Applies the .paper-tex texture,
// the shadow-paper drop, and optionally a three-hole punch down the
// left side (for bound folders). Use for mobile dossier pages,
// tribunal ballots, any document.

import type { CSSProperties, ReactNode } from "react";
import { rpColors } from "@replicant/tokens";

export interface RPPaperProps {
  children?: ReactNode;
  style?: CSSProperties;
  withHoles?: boolean;
  /** Rotation in degrees for a slightly-off paper feel. */
  rotate?: number;
  tone?: "paper" | "pink" | "bright";
}

export function RPPaper({
  children,
  style,
  withHoles = false,
  rotate = 0,
  tone = "paper",
}: RPPaperProps) {
  const bg = tone === "pink" ? rpColors.paper2 : tone === "bright" ? rpColors.paper3 : rpColors.paper;
  return (
    <div
      className="paper-tex"
      style={{
        background: bg,
        boxShadow:
          "0 1px 0 rgba(0,0,0,0.04), 0 2px 6px rgba(60,50,30,0.18), 0 12px 24px rgba(60,50,30,0.08)",
        position: "relative",
        padding: 24,
        transform: rotate ? `rotate(${rotate}deg)` : "none",
        ...style,
      }}
    >
      {withHoles && (
        <div
          aria-hidden="true"
          style={{
            position: "absolute",
            left: 8,
            top: 20,
            bottom: 20,
            display: "flex",
            flexDirection: "column",
            justifyContent: "space-around",
            alignItems: "center",
          }}
        >
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              style={{
                width: 12,
                height: 12,
                borderRadius: "50%",
                background: "rgba(0,0,0,0.18)",
                boxShadow: "inset 1px 1px 2px rgba(0,0,0,0.25)",
              }}
            />
          ))}
        </div>
      )}
      {children}
    </div>
  );
}
