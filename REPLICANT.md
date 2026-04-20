# Replicant

A real-world social-deduction party game (Humans vs. AI) with a TV host display + mobile phones.

This repo is mid-pivot from an earlier Secret Hitler prototype. The Go engine at the repo root (`cmd/api/`, `handlers/game/`, `models/`, `schemas/secrethitler/`) is intact and will be wired up as the realtime backend in Phase 2 — semantics already renamed to human / ai / rogue. The new monorepo (`apps/`, `packages/`) lives alongside it.

## Status

- Phase 1 — Foundation · ✅ complete
- Phase 2 — Session lifecycle · ✅ complete
- Phase 3 — Voting + chat · ✅ complete
- **Phase 4 — Narrator (LLM + TTS) · ✅ complete (demo-ready)**
- Phase 5 — Night cycle + win states · pending

## Layout

```
replicant/
├─ apps/
│  └─ host/               Next.js App Router (TV / laptop display)
├─ packages/
│  ├─ tokens/             CSS variables + utility classes + TS color constants
│  └─ ui/                 RP* component library (shared between host + mobile)
├─ design_handoff_replicant/   Design reference — see README.md there
│
├─ cmd/api/               Go server entrypoint (will become apps/server in Phase 2)
├─ handlers/game/         Game engine (phase machine, rules, powers)
├─ models/                DynamoDB-backed entities
├─ schemas/secrethitler/  Canonical enums (now human/ai/rogue)
└─ api/                   HTTP + SSE transport
```

## Phase 1 — what's working

### `packages/tokens`
- Complete port of `design_handoff_replicant/tokens.css` → `packages/tokens/src/styles.css`
- All CSS variables (paper, ink, broadcast, stamps, cyan, amber, spacing, shadows)
- Utility classes: `.paper-tex`, `.broadcast-tex`, `.scanlines`, `.stamp`, `.redacted`, `.perf-top`, `.t-eyebrow`, `.t-display`, `.t-memo`
- Keyframes ported: `blink`, `stampIn`, `tickerFeed`, `flicker`
- Keyframes added per handoff recommendations: `paperSlide`, `redactionSweep`
- Font family vars resolve to `next/font` CSS variables (see `apps/host/app/layout.tsx`), falling back to Google-named families
- TS constants in `packages/tokens/src/index.ts`: `rpColors`, `rpSpace`, `rpFonts`

### `packages/ui`
All 13 primitives from the Phase 1 checklist, ported as typed React components:

| Component | Notes |
|---|---|
| `RPSeal` | SVG seal, text-on-path, mix-blend-mode multiply |
| `RPWordmark` | Offset ghost duplicate |
| `RPStamp` | 3 variants (red / green / blue), `animate` prop plays stampIn keyframe |
| `RPRedact` | `animate` prop sweeps the bar in from left (redactionSweep) |
| `RPMemoHeader` | Masthead + form metadata bar |
| `RPButton` | 4 variants (primary / stamp / ghost / quiet), press-state translate |
| `RPPlayerChip` | Two sizes, all statuses (ALIVE / ACTIVE / TERMINATED / JOINING) |
| `RPTimer` | Mono countdown + blinking square, `danger` variant |
| `RPCheckbox` | Typewriter `X` mark (or custom), checked/unchecked |
| `RPTicker` | Marquee strip, seamless loop |
| `RPPaper` | 3 tones + optional 3-hole punch |
| `RPTVChrome` | Broadcast frame with REC header + ticker footer + corner reg marks |
| `RPPhone` | Phone bezel wrapper |
| `RPMobileStatusBar` | Fake iOS status bar |

### `apps/host`
- Next.js 14 App Router
- Fonts loaded via `next/font/google`: Oswald (500/600/700), Special Elite, JetBrains Mono (400/500/700), Inter Tight (400–700)
- Global token stylesheet imported in root layout
- **`/`** — placeholder with wordmark + link to `/dev`
- **`/dev`** — component library showcase. 8 sections, every primitive in every useful variant, paper canvases + broadcast canvases side by side, motion studies live

## How to run

```bash
# from the repo root — first time only
pnpm install

# dev server (http://localhost:3000)
pnpm dev:host

# production build
pnpm --filter @replicant/host build

# typecheck everything
pnpm -r typecheck
```

Open http://localhost:3000/dev to see the component library checkpoint.

## Architectural decisions worth noting

**We kept the Go backend instead of scaffolding Next.js API routes.** The handoff's default architecture is Next API routes or Fastify. We have a complete tested Go engine (`handlers/game/`) — phase state machine, executive powers, veto, 360-game action matrix. Rewriting it in TS would throw away weeks of iteration. In Phase 2 we'll wrap it with a session/realtime layer and expose it as `apps/server` (still running under `cmd/api`). The Zod schemas in `packages/schema` will mirror the Go types.

**`packages/ui` uses "use client" narrowly.** Only `RPButton` needs it (useState for press state). All other primitives are pure and render fine in Server Components. This keeps the default bundle small.

**CSS is plain CSS, not CSS-in-JS.** The tokens file lives in `packages/tokens/src/styles.css` and is imported once in the root layout. Components use inline styles + utility class names — matches the handoff's approach and avoids a runtime styling dep.

## Phase 2 — what's working

### `packages/schema`
Zod mirror of the Go engine types. Single source of truth for every
wire value the apps see.
- Enums: `Role`, `Party`, `PolicyType`, `GameStatus`, `GamePhase`, `VoteChoice`, `GovernmentStatus`, `ExecutiveActionType`, `WinCondition`, `EventType`, `ProgressReason`, `AudienceScope`, `DeviceRole`
- Models: `Game` / `Session`, `Player`, `Government`, `Vote`, `ChannelMsg` (stub for Phase 3)
- Wire: `Envelope`, `GameEvent`, `Audience`, and typed payloads for every event (game_created … game_ended + `PhaseChangedPayload`)

### `apps/server` (Go, pre-existing)
- New CORS middleware (`api/server.go`) — env-gated via `CORS_ALLOWED_ORIGINS`; not needed in dev (Next rewrites), present for prod deploys.
- Everything else unchanged — lobby creation, role allocation, SSE streams, simulator.

### `apps/host`
- **`/`** — boot screen. Checks cached session, offers **OPEN NEW INTAKE** → creates a board-only session via `POST /api/v1/games` and routes to `/lobby/CODE`.
- **`/lobby/[code]`** — HostLobby ported pixel-close from the handoff. Wordmark, REPLICANT ghost dup, instruction block, session code + **live `QRCodeSVG`** pointing at `http://<host>:3001/join/CODE`, 2-column roster grid that fills live as `player_joined` envelopes arrive, DEPLOY ▸ button (enabled once ≥ 5 players). DEPLOY calls `POST /host/start`, then the `game_started` envelope auto-routes to `/game/CODE`.
- **`/game/[code]`** — Phase-2 placeholder board. Renders current phase + round + seat list. Phase 3 replaces this with HostResults + narrator screens as gameplay events drive the UI.
- All pages run inside `RPTVChrome` so they share broadcast chrome + ticker footer + corner registration marks.
- `lib/api.ts` / `lib/session.ts` / `lib/useStream.ts` / `lib/deviceId.ts` — typed fetch client, localStorage session cache, SSE hook, per-device token.

### `apps/mobile` (new Next.js PWA on :3001)
- Mirrors the host's `lib/` structure (api, session, deviceId, useStream) for its own role.
- **`/`** — citizen entry. 6-char code input → routes to `/join/CODE`. Bounces straight to `/game` if a session is already cached.
- **`/join/[code]`** — join form reached via QR or manual entry. Code is locked, player types display name → `POST /api/v1/games/join` → caches session → `/game`.
- **`/game`** — in-game dossier. Renders one of two panels based on SSE state:
  - **Waiting**: status `lobby` with no role yet — "AWAITING DEPLOYMENT" on paper.
  - **RoleReveal**: once the `roles_assigned` whisper for this player arrives, shows the full MobileRoleReveal port (classification header, ghost-dup role wordmark, typewriter description with a redacted phrase, CLASSIFIED stamp, Known Kin chip list for replicants with visible teammates, objective block, "I ACCEPT" button). Roles / party / teammates persist in the session cache so a refresh lands back on the reveal.
- Role copy is Replicant-flavored: HUMAN (stamp-blue), REPLICANT (stamp-red), PRIME (amber).

## Two-device demo

```bash
# Terminal 1 — Go engine
docker compose up -d dynamodb      # one-time
./scripts/create-table.sh          # one-time (or curl shortcut — see create-table.json)
pnpm dev:server                    # :8080

# Terminal 2 — host TV
pnpm dev:host                      # :3000

# Terminal 3 — mobile PWA
pnpm dev:mobile                    # :3001
```

Then:
1. **Host device** (TV / laptop): open `http://localhost:3000` → **OPEN NEW INTAKE** → the lobby appears with a session code and QR.
2. **Mobile device** (phone on same LAN — or second browser window): scan the QR (`http://<host-ip>:3001/join/CODE`) or go to `http://localhost:3001` and type the code → enter a display name → lands on **AWAITING DEPLOYMENT**.
3. Repeat step 2 until 5+ citizens are seated. The host TV's roster updates live via SSE.
4. Host clicks **DEPLOY ▸** → Go engine rolls roles → mobile SSEs receive their private `roles_assigned` whisper → each phone flips from Waiting to **ROLE ASSIGNMENT** with classification, description, and Known Kin (replicants only).

If the QR target is unreachable from the phone (common when dev machine is on a firewall), override the QR URL with `NEXT_PUBLIC_MOBILE_URL=https://your.ngrok.io` at `pnpm dev:host` time.

## Phase 3 — what's working

### `apps/server` (Go)
- New `SendChat` engine method in `handlers/game/chat.go` — scopes by channel audience (`ai` = every alive AI/Rogue) and emits one whisper per member.
- New endpoint `POST /api/v1/games/:gameId/player/chat` wired on the player route group.
- New envelope `chat_message` + schema additions (`ChatChannel` enum, `ChatMessagePayload`).

### `apps/mobile` — full tribunal loop
Phase-dispatched action panels render based on `game.phase` + role + whisper state:
- **NominatePanel** — candidate picker with RPCheckbox rows, "▸ NOMINATE SAM" stamp button. Shown to the president during `nomination`.
- **VotePanel** — Ja/Nein ballot with the proposed government card. Shown to everyone alive during `election`.
- **DiscardPanel** — 3 policy buttons (colored by type) for the president during `legislative_president`.
- **EnactPanel** — 2 policy buttons + optional "Propose veto" for the chancellor during `legislative_chancellor`.
- **VetoResolvePanel** — Accept / Reject ja-nein during `veto_requested`, president only.
- **ExecutivePanel** — adapts to power: Investigate / Special Election / Execution show the target picker (with inline investigation-result tags); Policy Peek shows the three upcoming policies.
- **Terminated** — full-screen "PROCESSING COMPLETE" with animated stamp if the player has been executed mid-game.
- **GameOverPanel** — winner announcement + per-condition memo + personal-file-reflection stamp.
- **WaitFor** — used for every "someone else is acting" state, with a memo quoting the Committee.
- **RoleReminder** — sticky bottom bar with a hold-to-reveal trigger, always available once the dossier is acknowledged.
- **ChatDrawer** — AI-faction private room (70vh bottom sheet). Typewriter bubbles, self-alignment, archive-on-close micro-copy. Floating red **◼ KIN CHANNEL** FAB toggles it; unread count badge shows on the FAB.

Mobile API surface extended: `nominateChancellor`, `castVote`, `presidentDiscard`, `chancellorEnact`, `proposeVeto`, `resolveVeto`, `executeAction`, `sendChat`.

### `apps/host` — phase-aware TV broadcast
Replaces the Phase-2 placeholder. Layout is two columns inside `RPTVChrome`:
- **Left stage**: phase-specific `BigLabel` — NOMINATION / TRIBUNAL · VOTE / SORTING ROOM / DRAFTING FLOOR / VETO CONSIDERATION / EXECUTIVE ORDER — with ghost-dup title treatment, Committee-voice memo, and a live `VoteMeter` bar during election phase showing ballots cast vs alive count.
- **Last-election banner**: persists after every real election result — "LAST TRIBUNAL · NAME + NAME · JA X NEIN Y · PASSED/REJECTED" in mono with an accent color.
- **Right column**:
  - **PolicyTrack** — HUMAN 5-slot / AI 6-slot bars filling with each enactment, plus a 3-pip election-tracker strip and VETO LOCKED/UNLOCKED indicator.
  - **Assembly** — every seated player with portrait, subject number, name, and live badges (P for president, C for chancellor, ✖ for terminated). Dead players render with crimson border and reduced opacity.
- **TerminatedOverlay** — full-screen fade + animated `TERMINATED` stamp + "NAME · has been selected for TERMINATION" typewriter line. Dismisses automatically after 4 s. If the executed player was the Prime, the overlay also reveals that.
- **WinnerPanel** — takes over the chrome on `game_ended`. 200px display type, ghost-dup, condition-specific memo, population status footer.

### End-to-end demo flow
1. Host → Deploy in the lobby.
2. Mobile phones flip to role reveal → **I ACCEPT** → concealed card + hold-to-reveal.
3. Host TV shows **NOMINATION** · "X is selecting a Chancellor".
4. President phone: NominatePanel → selects Chancellor → TV flips to **TRIBUNAL · VOTE** with live ballot meter.
5. Every alive phone: VotePanel (ja/nein). TV meter fills; once the last vote is in, engine resolves and the banner snaps to PASSED/REJECTED.
6. On pass: president sees DiscardPanel (3 policies). TV shows SORTING ROOM. Then chancellor sees EnactPanel (2 policies, optional veto). TV shows DRAFTING FLOOR.
7. If a fascist policy triggers a power: president sees ExecutivePanel, TV shows EXECUTIVE ORDER. Investigate → inline result on the candidate row. Execution → on confirmation, TV fires TerminatedOverlay for 4 s.
8. Game-end (5 human / 6 AI / Prime elected / Prime executed) → host WinnerPanel + mobile GameOverPanel.

### AI cabal chat
Every AI/rogue phone shows the KIN CHANNEL FAB during play. Tapping opens the drawer; messages whisper only to the AI faction (dead members lose access). Replies stream in live via `chat_message` envelopes.

## Phase 4 — what's working

### `apps/server`
- New `handlers/narrator/` package: `Narrator{Speak(cue)}` calls the foundation's Anthropic connector for script generation and the ElevenLabs TTS connector for audio. System prompt and per-cue templates in `prompts.go` use the voice/copy examples from the design handoff verbatim as few-shot.
- New endpoint `POST /api/v1/games/:gameId/host/narrate` (host-only): accepts `{cue, vars?, text?}`, returns `{cueId, script, audioUrl, audioId, format, tookMs}`. Also broadcasts a `narrator_speak` envelope on the game's hub so every subscriber receives the script + a `data:audio/mpeg;base64,...` URL.
- New endpoint `GET /api/v1/narrator/audio/:cueId` (auth required): serves cached MP3 bytes for the cue, with 5-minute in-memory TTL — useful for re-fetch / debugging.
- Cue catalogue: `opening`, `election_passed`, `election_rejected`, `execution`, `closing`, `custom`. Custom skips the LLM and pipes raw text straight into TTS.
- Construction in `cmd/api/main.go:buildNarrator` — both `ANTHROPIC_API_KEY` and `ELEVENLABS_API_KEY` must be set or the narrator route returns 503 (server logs why on startup).
- Voice rules baked into the system prompt: never an exclamation mark, never an emoji, "citizen / subject / assembly / processed / regrettably" vocabulary, dry-humor allowed sparingly. Output also `sanitize`d server-side as a belt-and-suspenders guard against stray punctuation.

### `apps/host` — `HostNarrator` overlay
- `apps/host/app/game/[code]/narrator.tsx`: full-bleed overlay on top of the broadcast chrome, plays `<audio>` driven by the data-URL payload, routes the audio through a Web Audio `AnalyserNode` (FFT 256, 128 frequency bins) to drive an **80-bar waveform** with first 60 in cyan, rest in dim broadcast-rule. Bars update once per `requestAnimationFrame` from real audio amplitude — true reactive visualization, not a faked sine.
- Typewriter reveal of the script in 48px Special Elite, with a cursor block that blinks until the line completes. The last all-caps reveal word (4+ letters, e.g. `HUMAN` / `REPLICANT`) is automatically given the stamp-red underline accent from the handoff.
- Transport row: cue id + ORACLE-III synth tag on the left, ⏭ CONTINUE skip button on the right.
- `▸ SPEAK` button on the host TV (cyan, bottom-right) — host can manually trigger an opening cue. Disabled while the narrator is speaking, hidden after game ends.
- Subscribes to the `narrator_speak` envelope on the board stream; whenever one arrives the overlay renders, plays, and self-dismisses on `<audio> ended` (with a 650 ms settle so the typewriter line isn't truncated).

### `packages/schema` — additions
- New `narrator_speak` event type and `NarratorSpeakPayload` Zod schema (`cueId`, `cue`, `script`, `audioUrl`, `audioId`, `format`, `tookMs`).

### How to demo

1. Set in backend `.env` (both required):
   ```
   ANTHROPIC_API_KEY=sk-ant-...
   ELEVENLABS_API_KEY=sk_...
   ELEVENLABS_VOICE_ID=...
   # optional: ELEVENLABS_MODEL_ID=eleven_turbo_v2_5
   # optional: NARRATOR_MODEL=claude-sonnet-4-6
   ```
2. Restart `pnpm dev:server`. Logs should print `narrator ready model=claude-sonnet-4-6 voice=...`.
3. Spin up host + mobile, run a quick lobby + start.
4. On the host TV, click **▸ SPEAK** in the bottom-right corner.
5. ~1-3s later the overlay slides in: 80-bar waveform pumping in cyan against the audio, typewriter script reveals as the Committee speaks, last-word reveal underlined in stamp-red.
6. ⏭ CONTINUE dismisses early; otherwise the overlay self-clears when audio ends.

If the narrator route returns 503, the server is missing one of the two creds — check `pnpm dev:server` startup logs for the disabled reason.

## Phase 5 — what's next

- `HostNight` lights-out screen + 90 s timer + faction-instruction grid
- Per-player Night Cycle task minigame (humans complete a small puzzle, replicants pick a target, singularity intervenes)
- Rich win-state reveal screens replacing the current banners — TERMINATED stamp on the Prime, "Singularity outlasts" amber finale, etc.
- Auto-cued narrator on round transitions + executions (already plumbed via `narrator_speak`; just needs trigger sites in the engine)
- Spectator / terminated dashboard on host TV
- End-to-end playable session for 5+ people across LAN
