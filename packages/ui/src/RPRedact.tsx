// RPRedact — solid-black censoring bar. Renders the text underneath
// (for accessibility + copy-paste) but paints it ink-on-ink so it's
// invisible. Wrap `.animate-redaction` on a child via the wrapper prop
// to sweep the bar in from the left.

import type { CSSProperties, ReactNode } from "react";

export interface RPRedactProps {
  children: ReactNode;
  /** Optional fixed width. Useful when the underlying text is short. */
  width?: number | string;
  /** Play the redaction-sweep keyframe on mount. */
  animate?: boolean;
  style?: CSSProperties;
}

export function RPRedact({ children, width, animate = false, style }: RPRedactProps) {
  if (!animate) {
    return (
      <span className="redacted" style={{ display: "inline-block", width, ...style }}>
        {children}
      </span>
    );
  }
  // When animating, wrap the redaction bar in a clip so the sweep
  // happens over the still-visible text and then covers it.
  return (
    <span
      style={{
        position: "relative",
        display: "inline-block",
        width,
        color: "var(--ink)",
        ...style,
      }}
    >
      <span style={{ visibility: "hidden" }}>{children}</span>
      <span
        aria-hidden="true"
        className="redacted animate-redaction"
        style={{
          position: "absolute",
          inset: 0,
          display: "inline-block",
        }}
      />
    </span>
  );
}
