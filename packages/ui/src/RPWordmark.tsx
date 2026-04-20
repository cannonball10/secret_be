// RPWordmark — "REPLICANT" with a ghost duplicate offset behind.
// The visual pun: you never see just one replicant.

import { rpColors } from "@replicant/tokens";

export interface RPWordmarkProps {
  size?: number;
  color?: string;
  /** Color of the offset ghost duplicate. */
  ghost?: string;
  /** Override the displayed word. Defaults to "Replicant". */
  children?: string;
}

export function RPWordmark({
  size = 72,
  color = rpColors.ink,
  ghost = rpColors.stampRed,
  children = "Replicant",
}: RPWordmarkProps) {
  return (
    <div
      style={{
        position: "relative",
        display: "inline-block",
        fontFamily: "var(--font-display)",
        fontWeight: 700,
        fontSize: size,
        letterSpacing: "-0.01em",
        lineHeight: 0.9,
        textTransform: "uppercase",
      }}
    >
      <span
        aria-hidden="true"
        style={{
          position: "absolute",
          left: size * 0.04,
          top: size * 0.04,
          color: ghost,
          opacity: 0.35,
          mixBlendMode: "multiply",
        }}
      >
        {children}
      </span>
      <span style={{ position: "relative", color }}>{children}</span>
    </div>
  );
}
