// RPSeal — the Department of Human Affairs stamp.
//
// Circular mark: two concentric rings, inner monogram "R" over a
// tiny "REPLICANT" label, text-on-path around the ring reading
// "DEPT. OF HUMAN AFFAIRS · FORM R-07 · CLASSIFIED ·". Rendered with
// mix-blend-mode: multiply so it bleeds into the surface like ink.

import { useId } from "react";
import { rpColors, rpFonts } from "@replicant/tokens";

export interface RPSealProps {
  size?: number;
  color?: string;
  /** Rotation in degrees. Default -6 — stamp printed slightly askew. */
  rotate?: number;
}

export function RPSeal({ size = 120, color = rpColors.stampRed, rotate = -6 }: RPSealProps) {
  const id = useId();
  const pathId = `seal-path-${id}`;
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 120 120"
      style={{ transform: `rotate(${rotate}deg)`, mixBlendMode: "multiply", opacity: 0.9 }}
      aria-label="Department of Human Affairs seal"
    >
      <defs>
        <path
          id={pathId}
          d="M 60,60 m -44,0 a 44,44 0 1,1 88,0 a 44,44 0 1,1 -88,0"
        />
      </defs>
      <circle cx="60" cy="60" r="54" fill="none" stroke={color} strokeWidth="2" />
      <circle cx="60" cy="60" r="49" fill="none" stroke={color} strokeWidth="1" />
      <circle cx="60" cy="60" r="36" fill="none" stroke={color} strokeWidth="1.5" />
      <text
        fontFamily={rpFonts.display}
        fontWeight="700"
        fontSize="9"
        letterSpacing="2"
        fill={color}
      >
        <textPath href={`#${pathId}`} startOffset="2%">
          DEPT. OF HUMAN AFFAIRS · FORM R-07 · CLASSIFIED ·
        </textPath>
      </text>
      <text
        x="60"
        y="58"
        textAnchor="middle"
        fontFamily={rpFonts.display}
        fontWeight="700"
        fontSize="30"
        fill={color}
        letterSpacing="-1"
      >
        R
      </text>
      <text
        x="60"
        y="74"
        textAnchor="middle"
        fontFamily={rpFonts.mono}
        fontSize="6"
        fill={color}
        letterSpacing="2"
      >
        REPLICANT
      </text>
      <line x1="28" y1="60" x2="40" y2="60" stroke={color} strokeWidth="1" />
      <line x1="80" y1="60" x2="92" y2="60" stroke={color} strokeWidth="1" />
    </svg>
  );
}
