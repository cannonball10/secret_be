// RPCheckbox — form ballot box. The "X" is drawn in Special Elite
// (typewriter) in stamp-red — same as a paper ballot punched with ink.

import { rpColors } from "@replicant/tokens";

export interface RPCheckboxProps {
  checked?: boolean;
  mark?: string;
  size?: number;
  color?: string;
}

export function RPCheckbox({
  checked = false,
  mark = "X",
  size = 22,
  color = rpColors.stampRed,
}: RPCheckboxProps) {
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        width: size,
        height: size,
        border: `2px solid ${rpColors.ink}`,
        background: rpColors.paper3,
        fontFamily: "var(--font-typewriter)",
        fontSize: size * 0.85,
        lineHeight: 1,
        color,
        fontWeight: 700,
      }}
      role="checkbox"
      aria-checked={checked}
    >
      {/* Inner span nudges the glyph toward optical center. Special
          Elite's ascent metric hangs glyphs high in the line box, so
          flex centering alone lands them visibly above middle. A tiny
          percentage translate scales with font size and fixes it. */}
      <span
        aria-hidden="true"
        style={{
          display: "block",
          transform: "translateY(7%)",
        }}
      >
        {checked ? mark : ""}
      </span>
    </span>
  );
}
