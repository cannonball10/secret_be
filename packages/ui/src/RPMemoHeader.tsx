// RPMemoHeader — a bureaucratic document header strip. Two-line
// masthead (department + title) on the left, form metadata
// (form number + classification) on the right, separated by a 2px
// ink rule. Sits at the top of any RPPaper used as a formal memo.

import { rpColors } from "@replicant/tokens";

export interface RPMemoHeaderProps {
  title?: string;
  /** Form number, rendered mono on the right. */
  no?: string;
  classification?: string;
}

export function RPMemoHeader({
  title = "INTERNAL MEMO",
  no = "R-07-2186",
  classification = "CLASSIFIED",
}: RPMemoHeaderProps) {
  return (
    <div
      style={{
        borderBottom: `2px solid ${rpColors.ink}`,
        paddingBottom: 10,
        marginBottom: 16,
        display: "flex",
        justifyContent: "space-between",
        alignItems: "flex-end",
        gap: 12,
      }}
    >
      <div>
        <div className="t-eyebrow" style={{ color: rpColors.inkFaded, marginBottom: 2 }}>
          DEPT. OF HUMAN AFFAIRS
        </div>
        <div className="t-display" style={{ fontSize: 22, color: rpColors.ink }}>
          {title}
        </div>
      </div>
      <div
        style={{
          textAlign: "right",
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          color: rpColors.inkFaded,
          lineHeight: 1.5,
        }}
      >
        <div>FORM № {no}</div>
        <div style={{ color: rpColors.stampRed }}>■ {classification}</div>
      </div>
    </div>
  );
}
