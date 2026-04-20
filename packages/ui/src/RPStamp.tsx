// RPStamp — thump-on-paper bureaucratic stamp ("CLASSIFIED",
// "APPROVED", "TERMINATED", etc.). Uses the .stamp class from tokens
// for noise texture + mix-blend-mode. Pass `animate` to play the
// stampIn keyframe.

import type { CSSProperties, ReactNode } from "react";

export type RPStampVariant = "red" | "green" | "blue";

export interface RPStampProps {
  children: ReactNode;
  variant?: RPStampVariant;
  /** Rotation in degrees. Also used by the stampIn animation. */
  rotate?: number;
  /** Font size of the stamp text in px. */
  size?: number;
  /** If true, play the stamp-in keyframe on mount. */
  animate?: boolean;
  style?: CSSProperties;
}

export function RPStamp({
  children,
  variant = "red",
  rotate = -6,
  size = 22,
  animate = false,
  style,
}: RPStampProps) {
  const variantClass =
    variant === "green" ? " stamp--green" : variant === "blue" ? " stamp--blue" : "";
  const animateClass = animate ? " animate-stamp" : "";
  const transform = animate ? undefined : `rotate(${rotate}deg)`;
  const cssVars: CSSProperties = {
    // Used by the stampIn keyframe so rotation survives the scale anim.
    ["--stamp-rotate" as string]: `${rotate}deg`,
  };
  return (
    <span
      className={`stamp${variantClass}${animateClass}`}
      style={{
        fontSize: size,
        transform,
        ...cssVars,
        ...style,
      }}
    >
      {children}
    </span>
  );
}
