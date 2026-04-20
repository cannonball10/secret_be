"use client";

// RPButton — stamp-on-paper action button.
//
// Four variants from the handoff:
//   primary — solid ink on paper-3
//   stamp   — stamp-red, official call-to-action
//   ghost   — outlined ink
//   quiet   — paper-2 chip for secondary actions
//
// Press state: translates the button 2px down-right and collapses the
// hard drop-shadow, so it reads as a physical stamp being pressed.

import type { ButtonHTMLAttributes, CSSProperties, ReactNode } from "react";
import { useState } from "react";
import { rpColors } from "@replicant/tokens";

export type RPButtonVariant = "primary" | "stamp" | "ghost" | "quiet";

export interface RPButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
  children: ReactNode;
  variant?: RPButtonVariant;
  icon?: ReactNode;
  /** Fill parent width. */
  full?: boolean;
}

const variants: Record<RPButtonVariant, { bg: string; fg: string; border: string }> = {
  primary: { bg: rpColors.ink, fg: rpColors.paper3, border: rpColors.ink },
  stamp: { bg: rpColors.stampRed, fg: "#F2EAD3", border: rpColors.stampRed2 },
  ghost: { bg: "transparent", fg: rpColors.ink, border: rpColors.ink },
  quiet: { bg: rpColors.paper2, fg: rpColors.ink, border: rpColors.ink },
};

export function RPButton({
  children,
  variant = "primary",
  icon,
  full = false,
  style,
  disabled,
  onMouseDown,
  onMouseUp,
  onMouseLeave,
  ...rest
}: RPButtonProps) {
  const [pressed, setPressed] = useState(false);
  const s = variants[variant];
  const base: CSSProperties = {
    background: s.bg,
    color: s.fg,
    border: `2px solid ${s.border}`,
    fontFamily: "var(--font-display)",
    fontSize: 15,
    fontWeight: 600,
    letterSpacing: "0.08em",
    textTransform: "uppercase",
    padding: "13px 22px",
    cursor: disabled ? "not-allowed" : "pointer",
    width: full ? "100%" : "auto",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: 10,
    opacity: disabled ? 0.45 : 1,
    boxShadow: pressed
      ? "1px 1px 0 rgba(28,26,21,0.25)"
      : "3px 3px 0 rgba(28,26,21,0.25)",
    transform: pressed ? "translate(2px,2px)" : "none",
    transition: "transform .08s, box-shadow .08s",
  };
  return (
    <button
      {...rest}
      disabled={disabled}
      style={{ ...base, ...style }}
      onMouseDown={(e) => {
        setPressed(true);
        onMouseDown?.(e);
      }}
      onMouseUp={(e) => {
        setPressed(false);
        onMouseUp?.(e);
      }}
      onMouseLeave={(e) => {
        setPressed(false);
        onMouseLeave?.(e);
      }}
    >
      {icon && <span style={{ fontSize: 16 }}>{icon}</span>}
      {children}
    </button>
  );
}
