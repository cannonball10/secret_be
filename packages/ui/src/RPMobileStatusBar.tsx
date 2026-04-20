// RPMobileStatusBar — minimal fake-iOS status bar for the mobile
// screens. Shows a fixed 21:47 so design screenshots stay stable;
// swap to real clock state later if we care.

import { rpColors } from "@replicant/tokens";

export function RPMobileStatusBar() {
  return (
    <div
      style={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        padding: "14px 28px 4px",
        fontFamily: "var(--font-mono)",
        fontSize: 12,
        color: rpColors.ink,
        fontWeight: 700,
      }}
    >
      <span>21:47</span>
      <span style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <span>●●●</span>
        <span>100%</span>
      </span>
    </div>
  );
}
