// RPTimer — mono countdown with a blinking square and an eyebrow label.

import { rpColors } from "@replicant/tokens";

export interface RPTimerProps {
  value?: string;
  label?: string;
  /** When true, recolors the dot + digits to stamp-red. */
  danger?: boolean;
}

export function RPTimer({ value = "02:14", label = "TIME REMAINING", danger = false }: RPTimerProps) {
  const c = danger ? rpColors.stampRed : rpColors.ink;
  return (
    <div style={{ display: "inline-flex", flexDirection: "column", alignItems: "flex-start", gap: 3 }}>
      <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
        {label}
      </div>
      <div style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
        <span
          className="animate-blink"
          style={{ width: 8, height: 8, background: c, display: "inline-block" }}
        />
        <span
          style={{
            fontFamily: "var(--font-mono)",
            fontWeight: 700,
            fontSize: 38,
            color: c,
            letterSpacing: "0.04em",
            lineHeight: 1,
          }}
        >
          {value}
        </span>
      </div>
    </div>
  );
}
