// RPPhone — phone bezel wrapper for mobile screens. Default
// dimensions (360×760) mirror a typical mid-size Android viewport.

import type { CSSProperties, ReactNode } from "react";
import { rpColors } from "@replicant/tokens";

export interface RPPhoneProps {
  children?: ReactNode;
  width?: number;
  height?: number;
  style?: CSSProperties;
}

export function RPPhone({ children, width = 360, height = 760, style }: RPPhoneProps) {
  return (
    <div
      style={{
        width: width + 16,
        height: height + 16,
        background: rpColors.ink,
        borderRadius: 44,
        padding: 8,
        boxShadow: "0 20px 50px rgba(0,0,0,0.2), inset 0 0 0 2px #2a2820",
        ...style,
      }}
    >
      <div
        style={{
          width,
          height,
          background: rpColors.paper,
          borderRadius: 36,
          overflow: "hidden",
          position: "relative",
        }}
      >
        <div
          aria-hidden="true"
          style={{
            position: "absolute",
            top: 10,
            left: "50%",
            transform: "translateX(-50%)",
            width: 110,
            height: 28,
            background: rpColors.ink,
            borderRadius: 20,
            zIndex: 10,
          }}
        />
        {children}
      </div>
    </div>
  );
}
