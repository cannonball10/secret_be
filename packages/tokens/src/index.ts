// @replicant/tokens — TS constants mirroring the CSS variables in
// styles.css. Prefer CSS `var(--paper)` etc. in stylesheets; use these
// constants only when you need a literal hex (e.g. inline SVG stroke,
// a color passed as a prop to a component that can't accept a var).

export const rpColors = {
  paper: "#E8DFC9",
  paper2: "#D9CFB5",
  paper3: "#F2EAD3",
  paperLine: "#B8AE93",

  ink: "#1C1A15",
  inkSoft: "#3A352A",
  inkFaded: "#6B6450",

  broadcast: "#12120E",
  broadcast2: "#1E1D17",
  broadcastRule: "#3A3A2E",

  stampRed: "#B32724",
  stampRed2: "#8E1A18",
  stampGreen: "#3F6B3A",
  stampBlue: "#24426B",

  cyan: "#5FD3CC",
  cyanSoft: "#2A6E6A",
  amber: "#D69E2E",
} as const;

export type RpColor = keyof typeof rpColors;

// Spacing scale — mirrors --s-1..--s-8.
export const rpSpace = {
  s1: 4,
  s2: 8,
  s3: 12,
  s4: 16,
  s5: 24,
  s6: 32,
  s7: 48,
  s8: 64,
} as const;

// Font family strings. Components that need a family literal (e.g.
// inline SVG text) can pull these; CSS should use `var(--font-*)`.
export const rpFonts = {
  display: "Oswald, Impact, sans-serif",
  sans: "'Inter Tight', -apple-system, 'Helvetica Neue', sans-serif",
  mono: "'JetBrains Mono', 'Courier New', monospace",
  typewriter: "'Special Elite', 'Courier New', monospace",
} as const;
