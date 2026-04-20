// RPTicker — marquee strip.
//
// Renders the concatenated items twice side-by-side and slides the
// track from 0 to -50%. The second copy fills the space the first
// vacates, producing a seamless loop without the old
// "paddingLeft: 100%" hack (which made the element 3× wider than it
// needed to be and caused subpixel jitter).

import { rpColors } from "@replicant/tokens";

export interface RPTickerProps {
  items?: string[];
  bg?: string;
  fg?: string;
  separator?: string;
  /** Seconds for the track to cycle once through itself. Default 28s
   *  — fast enough to feel live but slow enough to read each phrase
   *  in passing. Override per-call when a specific screen needs it. */
  durationSeconds?: number;
  /** Font size in px (or any CSS length). Defaults to 15px — readable
   *  from across a living room without dominating the broadcast. */
  fontSize?: number | string;
}

export function RPTicker({
  items = ["TRANSMISSION BEGINS", "REPLICANT PROTOCOL R-07", "CLASSIFIED"],
  bg = rpColors.ink,
  fg = rpColors.paper3,
  separator = "    ◼    ",
  durationSeconds = 28,
  fontSize = 15,
}: RPTickerProps) {
  // Trailing separator so the end of the text flows straight into
  // the start of the next loop instead of two words running together.
  const content = items.join(separator) + separator;
  return (
    <div
      style={{
        background: bg,
        color: fg,
        overflow: "hidden",
        padding: "10px 0",
        fontFamily: "var(--font-mono)",
        fontSize,
        letterSpacing: "0.22em",
        whiteSpace: "nowrap",
      }}
    >
      <div
        style={{
          display: "inline-flex",
          willChange: "transform",
          animation: `tickerFeed ${durationSeconds}s linear infinite`,
        }}
      >
        <span>{content}</span>
        <span aria-hidden="true">{content}</span>
      </div>
    </div>
  );
}
