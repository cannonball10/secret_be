# Handoff: Replicant — Design System & Prototype Screens

## Overview

**Replicant** is a social-deduction party game in the vein of Secret Hitler, Among Us, and Mafia — but played **in the real world** using:
- A **Host display** (TV / desktop / laptop) — driven by an AI narrator using LLM inference + TTS
- **Mobile devices** — each player uses their phone to see their role, vote, chat, and handle tasks

This handoff contains the **visual design system and key screen prototypes** for the game, rendered as HTML for reference.

**Theme**: Dystopian corporate bureaucracy. The AI Host is framed as "The Department of Human Affairs," which processes humanity's potential extinction as paperwork. Tone is playful and campy while staying quietly threatening — think *Papers Please* crossed with 1970s office horror.

---

## About the Design Files

The files in this bundle are **design references created in HTML** — prototypes showing intended look and behavior, not production code to copy directly.

Your task is to **recreate these HTML designs in the target codebase's environment** (React, Next.js, React Native, SwiftUI, etc.) using its established patterns and libraries. If no environment exists yet, choose the most appropriate stack for a real-time multiplayer game with:
- A browser-based Host display (stays on one device for the session)
- Mobile web or native app for players (lightweight, joinable via URL/QR)
- A realtime backend (websockets / Firebase / Supabase / Colyseus) for sync
- LLM inference + TTS for the AI narrator (e.g. Claude or GPT + ElevenLabs)

Recommended default: **Next.js (host) + React web PWA (mobile) + Supabase Realtime + server route for LLM/TTS**. Simple to deploy, easy to prototype.

---

## Fidelity

**High-fidelity.** Colors, typography, spacing, stamp treatments, and layouts are final and should be recreated pixel-close. The design system (`tokens.css`, `components.jsx`) is the source of truth for values.

Interactions are specified at prototype fidelity — the motion direction is settled (stamp thuds, paper slides, redaction reveals, typewriter text, waveform pulses) but exact timings are open to implementation taste.

---

## Brand System

### Concept
- **The Department of Human Affairs** is the in-world authority. All UI is its official paperwork.
- **Host display** = dark broadcast bulletin (surveillance, authority, the machine speaking).
- **Mobile display** = warm paper dossier (personal, confidential, citizen-facing).
- Same grid, same type, opposite materials.

### Wordmark
`REPLICANT` set in **Oswald 700**, uppercase, slightly negative tracking (`letter-spacing: -0.01em`).
A duplicate of the word sits behind the primary, offset by ~4% of the font size in both x and y, colored stamp-red at 35% opacity with `mix-blend-mode: multiply`. Visual pun — you never see just one "Replicant."

### Department Seal
Circular stamp, 2 concentric rings, inner monogram "R", text on path reading `DEPT. OF HUMAN AFFAIRS · FORM R-07 · CLASSIFIED ·`. Default rotation ~-4° to -6°, stamp-red ink, `mix-blend-mode: multiply`, opacity 0.9.

---

## Design Tokens

All tokens live in `tokens.css`. Summary:

### Colors

| Token | Hex | Purpose |
|---|---|---|
| `--paper` | `#E8DFC9` | Aged manila — default mobile surface |
| `--paper-2` | `#D9CFB5` | Carbon-copy pink-beige — secondary surface |
| `--paper-3` | `#F2EAD3` | Bright memo — cards, ballots |
| `--paper-line` | `#B8AE93` | Ruled lines, dashed dividers |
| `--ink` | `#1C1A15` | Carbon-ribbon black — text, borders |
| `--ink-soft` | `#3A352A` | Secondary text |
| `--ink-faded` | `#6B6450` | Tertiary text, mono labels |
| `--broadcast` | `#12120E` | Host TV base |
| `--broadcast-2` | `#1E1D17` | Host TV panel |
| `--broadcast-rule` | `#3A3A2E` | Host TV hairlines |
| `--stamp-red` | `#B32724` | Primary accent — danger, official stamp |
| `--stamp-red-2` | `#8E1A18` | Stamp shadow |
| `--stamp-green` | `#3F6B3A` | APPROVED stamp |
| `--stamp-blue` | `#24426B` | PROCESSED stamp |
| `--cyan` | `#5FD3CC` | Machine / AI signal (host only) |
| `--cyan-soft` | `#2A6E6A` | Cyan muted |
| `--amber` | `#D69E2E` | Caution / Singularity faction |

### Typography

| Role | Family | Use |
|---|---|---|
| Display | **Oswald** 500/600/700 | Headlines, wordmark, stamps, buttons |
| Typewriter | **Special Elite** 400 | The Department's voice, quotes, body narrative |
| Mono | **JetBrains Mono** 400/500/700 | Data, timers, form metadata, tickers |
| Sans | **Inter Tight** 400–700 | Minor UI, fallback |

All loaded from Google Fonts (see first line of `tokens.css`).

Typography utilities:
- `.t-eyebrow` — mono, 11px, 0.22em letter-spacing, uppercase
- `.t-display` — Oswald, uppercase, tight tracking
- `.t-memo` — Special Elite, slight positive tracking

### Spacing scale
`--s-1`..`--s-8` → 4, 8, 12, 16, 24, 32, 48, 64 px.

### Effects
- `.paper-tex` — multi-stop radial gradients + 2px repeating linear gradients on both axes to simulate fiber grain
- `.broadcast-tex` — scanline gradient + faint cyan ambient glow
- `.scanlines` — `::after` repeating 3px horizontal lines at `mix-blend-mode: multiply`, 0.5 opacity
- `.stamp` — 3px border, rotate, `mix-blend-mode: multiply`, opacity 0.88, with pseudo-element noise to look printed
- `.redacted` — solid black block over text, text color matches so content disappears

### Motion
- `blink` — 1s step-end, infinite (REC dot, cursors, status)
- `stampIn` — 0.6s `cubic-bezier(.2,1.2,.4,1)` — scale 3→1 with rotation-preserving transform, overshoot at 85%
- `tickerFeed` — 40s linear — marquee strips
- `flicker` — 4s — subtle CRT instability

Recommended additions when building the real app:
- Paper slide-in on screen transitions (translateY + fade)
- Redaction bar sweep (scaleX 0→1 from left) for role reveals
- Typewriter character-by-character reveal for narrator text

---

## Game Structure (reference — coordinate with design on gameplay)

### Factions & Roles
- **HUMAN** (majority) — Complete tasks. Deduce the Replicants. Vote to terminate.
- **REPLICANT** (minority) — Know each other. Sabotage & deceive. Vote a human each Night Cycle.
- **SINGULARITY** (solo, optional) — Third faction. Wins on chaos. Has a once-per-cycle special power.

### Phase Loop
1. **Intake** — players join via code/QR on host screen
2. **Role Allocation** — each phone shows its dossier
3. **Day / Discussion** — citizens talk (in person), AI narrator frames the round
4. **Tribunal / Voting** — phones show ballot; results revealed on host
5. **Night Cycle** — lights out. Humans do tasks, Replicants pick a target, Singularity acts
6. Loop until win condition met

### The AI Narrator
- Pure-typographic identity on the host display (no avatar / face).
- Renders as large typewriter-style quotes with a live waveform above.
- Backed by an LLM generating script + TTS for voice output.
- Personality: polite bureaucrat, slightly menacing, occasional dry humor. Never excited. Never uses emoji.

---

## Screens

All screens live in the prototype file `Replicant Design System.html`. Open it in a browser to see the full design canvas.

### Host Display (1920 × 1080) — `host-screens.jsx`

#### 1. `HostLobby` — Candidate Intake
**Purpose**: Pre-game lobby. Players join via session code.

**Layout**: 2-column grid, 50/50 split, full height.

Left column (padding `56px 72px`, gap 36):
- Eyebrow: "◼ DEPT. OF HUMAN AFFAIRS · FORM R-07" in cyan
- `REPLICANT` wordmark at 120px
- Subtitle line in typewriter font
- Intake instructions box (1px cyan border, translucent cyan bg, typewriter ordered list)
- Session code + QR row: mono code 64px in cyan with `text-shadow: 0 0 20px cyan66`; QR is 120×120 on paper

Right column:
- Header: `CANDIDATES REGISTERED` eyebrow + big fraction `06/08` in Oswald 72px
- Player roster: 2-col grid, each cell is a bordered box (cyan if ready, broadcast-rule if joining), 44×44 initial avatar + subject number + name + ● READY state
- Footer: typewriter quote + "DEPLOY ▸" cyan button

#### 2. `HostNarrator` — AI Transmission
**Purpose**: The narrator speaks.

**Layout**: Centered column, `padding: 40px 120px`.
- Eyebrow: "▸ AUDIO TRANSMISSION · SYNTHESIZED VOICE · NODE 04-7"
- Waveform: 80 bars × 6px wide, heights from `6 + |sin(i*0.4)*cos(i*0.17)|*54`, first 60 cyan, rest faded — bind to real audio amplitude in production
- Giant typewriter quote (54px), with key subjects underlined in stamp-red and the revealed identity word set in Oswald 72px cyan inline

Bottom row: transport controls (INTERRUPT / REPEAT / CONTINUE), buffer state, synth model tag.

#### 3. `HostNight` — Night Cycle
**Purpose**: Lights-out phase.

**Layout**: Vertical centered stack.
- Red eyebrow `▼ LIGHTS OUT ▼`
- `NIGHT` word at 240px with ghosted duplicate
- 3-column faction instructions grid (320px columns): HUMANS (cyan) / REPLICANTS (red) / SINGULARITY (amber) — each with eyebrow + typewriter instructions
- Bottom: Timer "01:30 CYCLE ENDS IN" + typewriter quote

#### 4. `HostResults` — Tribunal Result
**Purpose**: Vote resolution reveal.

**Layout**: 2-column grid (1.2fr / 1fr), `padding: 48px 80px`.

Left: Verdict column
- Subject number in Oswald 80px
- Player name at 180px stamp-red
- "…has been selected for TERMINATION." in typewriter
- Animated `TERMINATED` stamp — red 4px border, rotated -8°, positioned bottom-right, uses `stampIn` animation
- True identity reveal: label + `REPLICANT` word at 44px cyan + pipe + narrator quote

Right: Tally
- 4 rows, each: 56×56 avatar (red if eliminated) + name + count + bar
- Bar is `repeating-linear-gradient` 45° red hash if eliminated, solid cyan otherwise
- Footer: population status box with counts per faction

### Mobile (targeting 360×760 physical, iPhone/Android web) — `mobile-screens.jsx`

#### 5. `MobileRoleReveal`
- Status bar (mono)
- Header: subject number · day
- Intro block: eyebrow + "ROLE ASSIGNMENT" 42px + Department Seal 82px
- Role card (paper-3, 2px black border, 4px offset hard shadow):
  - Classification label
  - Role name `REPLICANT` at 64px stamp-red with ghosted duplicate
  - Typewriter description, contains `<span class="redacted">` for one phrase
  - Floating `CLASSIFIED` stamp (rotated 8°) at top-right
- Known Kin block: stamp-red left border 6px, shows fellow Replicant player chips
- Objective block in typewriter
- Footer: primary stamp button "I ACCEPT THE ASSIGNMENT ▸" + "HOLD TO REVEAL · RELEASE TO CONCEAL" microcopy

#### 6. `MobileVoting`
- Header + eyebrow "◼ OFFICIAL BALLOT · FORM V-02"
- "CAST YOUR VERDICT" 36px + typewriter instructions
- Timer strip: `0:47 BALLOT CLOSES` (danger red) + tally `CAST 05/08 · ABSTAIN 01`
- Candidate rows: scroll list. Each row:
  - Checkbox (22×22, black border, `X` mark in Special Elite red)
  - 42×42 avatar with faction color
  - Subject # + name
  - Self row shown at 55% opacity, disabled
  - Selected row: red 10px left border, paper-2 bg, shows current vote tally as red 5×16 pips on right
- Footer: stamp button "CAST BALLOT FOR SAM" + [ABSTAIN] / [CHANGE] links

#### 7. `MobileChat`
- Header with `ENC. AES-R7`
- Red banner: "▸ REPLICANT · PRIVATE" + "SWEEP IN 00:45" with blinking dot
- Message list:
  - Each message: 30×30 avatar + name + time + body bubble
  - User messages right-aligned, ink bg, paper-3 text, no border
  - Others left-aligned, paper-3 bg, 1.5px ink border, 2px offset shadow
  - Body in Special Elite 14px
  - System messages centered between dashed rules in mono
  - Typing indicator: eyebrow + 3 blinking red dots
- Input bar: ` ▸ ` prefix + typewriter input with blinking cursor block + SEND button
- Microcopy: "MESSAGES ARE ARCHIVED BY THE DEPARTMENT AFTER CYCLE END"

---

## Component Library — `components.jsx`

All components are global-scope React, prefix `RP*`, available on `window`:

- `RPSeal({size, color, rotate})` — circular department stamp (SVG)
- `RPWordmark({size, color, ghost})` — REPLICANT wordmark with ghosted dup
- `RPStamp({children, variant, rotate, size})` — "CLASSIFIED", "APPROVED", "TERMINATED" etc. `variant`: `red` | `green` | `blue`
- `RPRedact({children, width})` — solid-black censoring span
- `RPMemoHeader({title, no, classification})` — bureaucratic document header
- `RPButton({variant, full, icon, onClick})` — `variant`: `primary` (black) | `stamp` (red) | `ghost` | `quiet`. Offset hard-shadow press state.
- `RPPlayerChip({name, num, status, portrait, accent, small})` — citizen ID card
- `RPTimer({value, label, danger})` — mono countdown with blinking dot
- `RPTicker({items, bg, fg})` — marquee strip
- `RPPaper({children, withHoles, rotate, tone})` — document surface
- `RPCheckbox({checked, mark, size, color})` — ballot checkbox
- `RPTVChrome({title, nodeId, phase})` — host TV frame (header + scanlines + ticker footer + corner registration marks)
- `RPPhone({width, height})` — mobile device frame
- `RPMobileStatusBar` — fake iOS status bar

---

## Voice & Copy

The Department is:
- Polite, bureaucratic, quietly threatening
- Uses "citizen", "subject", "assembly", "processed", "regrettably"
- Never uses exclamation marks, emoji, or slang
- Slightly campy — allowed a dry joke ("Close your eyes. Or don't. We see everything regardless.")

Sample lines:
- "The Department appreciates your cooperation."
- "Subject #04 has been processed. You may return to your duties."
- "A satisfactory outcome. The paperwork reflects well on you."
- "Yesterday's termination of Subject MARA has been processed. Records indicate she was, regrettably, HUMAN."

Use these as few-shot examples when writing the LLM narrator system prompt.

---

## State Management (reference)

For the realtime game loop, recommended state shape per session:

```ts
Session {
  id: string
  code: string              // e.g. "R07-MKQ"
  phase: 'intake' | 'role_reveal' | 'day' | 'tribunal' | 'night' | 'results' | 'ended'
  dayNumber: number
  players: Player[]
  narratorScript: NarratorCue[]
  votes: Record<playerId, playerId | 'abstain'>
  chat: ChannelMsg[]        // per-faction channels
  tasks: TaskState[]        // per-player during night
}

Player {
  id, name, subjectNumber, portraitLetter, accentColor
  role: 'human' | 'replicant' | 'singularity'
  status: 'active' | 'terminated'
  deviceConnected: boolean
  knownKin: playerId[]      // for replicants
}
```

Sync via websocket / Supabase Realtime. Every phone subscribes to a filtered view of session state (role-gated).

---

## Assets

No image assets — everything is CSS, SVG, and fonts. Fonts loaded from Google Fonts (Oswald, Special Elite, JetBrains Mono, Inter Tight). When moving to a real app, self-host these for reliability.

QR code in the lobby is currently a placeholder pattern; use a real QR library (e.g. `qrcode.react`) in production with the session join URL.

Placeholder portraits are single-letter initials. Production can keep this or allow real avatars.

---

## Files in this bundle

- `README.md` — this document
- `tokens.css` — all design tokens, utility classes, keyframes
- `components.jsx` — shared component library (React, global-scope via Babel)
- `host-screens.jsx` — 4 host TV screens (Lobby, Narrator, Night, Results)
- `mobile-screens.jsx` — 3 mobile screens (Role Reveal, Voting, Chat)
- `Replicant Design System.html` — design canvas that composes everything; open in browser to see the whole system in one view

Open `Replicant Design System.html` first — it's the visual index of the entire system.

---

## Suggested next design passes (not yet built)

- Singularity-specific role reveal dossier (amber accent)
- Win-state reveals: Humans win / Replicants win / Singularity wins
- Night-phase task minigame UI (e.g. pattern-matching, form-filling, terminal sequences)
- Spectator / terminated view
- Host TV: role-reveal broadcast at game start
- Onboarding / first-time tutorial

Ask the designer (Claude in this thread) to produce these as additional prototypes when scoping the implementation.
