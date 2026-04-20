// Host TV — in-game broadcast.
//
// Phase-aware dispatch that mirrors the mobile side but from the
// Committee's point of view: no private whispers, persistent policy
// track + population stats, animated TERMINATED stamp when a delegate
// is expelled via executive power, and a winner banner at game end.

"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { RPStamp, RPTimer, RPTVChrome } from "@replicant/ui";
import { rpColors } from "@replicant/tokens";
import type {
  CableLeakedPayload,
  ChancellorNominatedPayload,
  ElectionResultPayload,
  Envelope,
  Game,
  GameStartedPayload,
  NarratorSpeakPayload,
  Party,
  Player,
  PlayerExecutedPayload,
  Role,
  VoteCastPayload,
  VoteChoice,
  WinCondition,
} from "@replicant/schema";
import { HostApi, ApiError } from "@/lib/api";
import { clearSession, loadSession } from "@/lib/session";
import { useStream } from "@/lib/useStream";
import { HostNarrator } from "./narrator";

interface LastElection {
  passed: boolean;
  jaVotes: number;
  neinVotes: number;
  presidentName: string;
  chancellorName: string;
}

interface Execution {
  playerName: string;
  countryName: string;
  wasRogue: boolean;
  at: number;
}

export default function HostGamePage() {
  const router = useRouter();
  const params = useParams<{ code: string }>();
  const code = (params?.code ?? "").toUpperCase();

  const [token, setToken] = useState<string | null>(null);
  const [game, setGame] = useState<Game | null>(null);
  const [players, setPlayers] = useState<Record<string, Player>>({});
  const [err, setErr] = useState<string | null>(null);

  const [presidentPlayerId, setPresidentPlayerId] = useState<string | null>(null);
  const [chancellorPlayerId, setChancellorPlayerId] = useState<string | null>(null);
  const [votedSet, setVotedSet] = useState<Record<string, true>>({});
  const [lastElection, setLastElection] = useState<LastElection | null>(null);
  const [execution, setExecution] = useState<Execution | null>(null);

  const [finalWinner, setFinalWinner] = useState<Party | null>(null);
  const [finalCondition, setFinalCondition] = useState<WinCondition | null>(null);

  // Narrator overlay state. A single payload at a time — if a new
  // narrator_speak arrives while one is already playing, it replaces
  // the current one so the host doesn't get behind.
  const [narratorCue, setNarratorCue] = useState<NarratorSpeakPayload | null>(null);
  const [narratorBusy, setNarratorBusy] = useState(false);

  // Cable leak overlay — the Committee's flagged-cable reveal. Shown
  // as a stacked card alongside the narrator. Cleared on the next
  // cable_phase_opened or when the narrator overlay completes.
  const [cableLeak, setCableLeak] = useState<CableLeakedPayload | null>(null);

  // Auto-cue guard — keyed on eventId so the same server event can't
  // fire the LLM twice (StrictMode double-invoke, re-render, etc.).
  const firedCuesRef = useRef<Set<string>>(new Set());

  // ── Boot ───────────────────────────────────────────────────────
  useEffect(() => {
    const cached = loadSession();
    if (!cached || cached.joinCode !== code) {
      router.replace("/");
      return;
    }
    setToken(cached.token);
    (async () => {
      try {
        const api = new HostApi({ token: cached.token });
        const snap = await api.getGame(cached.gameId);
        setGame(snap.game);
        const ix: Record<string, Player> = {};
        for (const p of snap.players) ix[p.playerId] = p;
        setPlayers(ix);
        if (snap.game.status === "lobby") {
          router.replace(`/lobby/${code}`);
        } else if (snap.game.status === "completed") {
          setFinalWinner(snap.game.winner ?? null);
          setFinalCondition(snap.game.winCondition ?? null);
        } else if (
          snap.game.status === "in_progress" &&
          snap.game.round === 1 &&
          snap.game.phase === "nomination" &&
          !firedCuesRef.current.has("opening")
        ) {
          // We just landed on the game page from the lobby. The real
          // game_started event fired before SSE opened, so fire the
          // opening cue here instead — fire-and-forget, silent on err.
          firedCuesRef.current.add("opening");
          try {
            await api.narrate(snap.game.gameId, "opening", {
              vars: { playerCount: String(snap.players.length) },
            });
          } catch {
            /* narrator disabled or errored — ignore */
          }
        }
      } catch (e) {
        setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
      }
    })();
  }, [code, router]);

  // Reconcile from the authoritative server snapshot. Fires on every
  // SSE open event so reconnections pick up any envelopes missed
  // during the gap (phase transitions, vote counts, policy tallies).
  const reconcileSnapshot = async () => {
    if (!token || !game) return;
    try {
      const api = new HostApi({ token });
      const snap = await api.getGame(game.gameId);
      setGame(snap.game);
      setPlayers((prev) => {
        const ix: Record<string, Player> = {};
        for (const p of snap.players) ix[p.playerId] = p;
        // Preserve any locally-seen isAlive=false in case the server
        // snapshot is a tick behind an execution envelope.
        for (const id of Object.keys(prev)) {
          if (prev[id] && !prev[id]!.isAlive && ix[id]) {
            ix[id] = { ...ix[id]!, isAlive: false };
          }
        }
        return ix;
      });
      if (snap.game.status === "completed") {
        setFinalWinner(snap.game.winner ?? null);
        setFinalCondition(snap.game.winCondition ?? null);
      }
    } catch {
      /* reconcile errors are non-fatal — the stream recovers on its own */
    }
  };

  // Fire a narrator cue without blocking the SSE handler. Silent on
  // failures — a missing ANTHROPIC/ELEVENLABS key just means the host
  // doesn't get voiceovers; the game continues.
  const fireCue = (cueKey: string, cue: string, vars: Record<string, string>) => {
    if (!token || !game) return;
    if (firedCuesRef.current.has(cueKey)) return;
    firedCuesRef.current.add(cueKey);
    void (async () => {
      try {
        const api = new HostApi({ token });
        await api.narrate(game.gameId, cue, { vars });
      } catch {
        /* narrator disabled or errored — ignore */
      }
    })();
  };

  // Reconcile only after an SSE error-then-reconnect — the initial
  // open is handled by the boot useEffect and shouldn't race with
  // concurrently-arriving envelopes.
  const hadStreamErrorRef = useRef(false);

  // ── SSE ────────────────────────────────────────────────────────
  useStream({
    gameId: game?.gameId ?? null,
    role: "board",
    token,
    onError: () => {
      hadStreamErrorRef.current = true;
    },
    onOpen: () => {
      if (hadStreamErrorRef.current) {
        hadStreamErrorRef.current = false;
        void reconcileSnapshot();
      }
    },
    onEnvelope: (env: Envelope) => {
      const type = env.event.type;

      if (type === "game_started") {
        const p = env.payload as GameStartedPayload | undefined;
        if (p) {
          fireCue("opening", "opening", {
            playerCount: String(p.playerCount),
          });
        }
      }

      if (type === "chancellor_nominated") {
        const p = env.payload as ChancellorNominatedPayload | undefined;
        if (!p) return;
        setPresidentPlayerId(p.presidentPlayerId);
        setChancellorPlayerId(p.chancellorPlayerId);
        setVotedSet({});
        setGame((prev) =>
          prev
            ? { ...prev, phase: "election", round: p.round, currentGovernmentId: p.governmentId }
            : prev,
        );
      }

      if (type === "vote_cast") {
        const p = env.payload as VoteCastPayload | undefined;
        if (!p) return;
        setVotedSet((prev) => ({ ...prev, [p.playerId]: true }));
      }

      if (type === "election_result") {
        // Real results carry `passed`; phase-change envelopes carry `to`/`from`.
        const p = env.payload as
          | (Partial<ElectionResultPayload> & { to?: Game["phase"]; from?: Game["phase"]; presidentSeat?: number })
          | undefined;
        if (!p) return;
        if (typeof p.passed === "boolean") {
          setLastElection({
            passed: p.passed,
            jaVotes: p.jaVotes ?? 0,
            neinVotes: p.neinVotes ?? 0,
            presidentName: presidentPlayerId ? players[presidentPlayerId]?.displayName ?? "" : "",
            chancellorName: chancellorPlayerId ? players[chancellorPlayerId]?.displayName ?? "" : "",
          });
        }
        if (p.to && p.from) {
          setGame((prev) =>
            prev
              ? {
                  ...prev,
                  phase: p.to!,
                  presidentSeat:
                    typeof p.presidentSeat === "number" ? p.presidentSeat : prev.presidentSeat,
                }
              : prev,
          );
        }
      }

      if (type === "chancellor_enacted") {
        const p = env.payload as { humanPoliciesEnacted?: number; aiPoliciesEnacted?: number } | undefined;
        if (!p) return;
        setGame((prev) =>
          prev
            ? {
                ...prev,
                humanPoliciesEnacted: p.humanPoliciesEnacted ?? prev.humanPoliciesEnacted,
                aiPoliciesEnacted: p.aiPoliciesEnacted ?? prev.aiPoliciesEnacted,
              }
            : prev,
        );
      }

      if (type === "top_deck_enacted") {
        const p = env.payload as { humanPoliciesEnacted?: number; aiPoliciesEnacted?: number } | undefined;
        if (!p) return;
        setGame((prev) =>
          prev
            ? {
                ...prev,
                humanPoliciesEnacted: p.humanPoliciesEnacted ?? prev.humanPoliciesEnacted,
                aiPoliciesEnacted: p.aiPoliciesEnacted ?? prev.aiPoliciesEnacted,
              }
            : prev,
        );
      }

      if (type === "player_executed") {
        const p = env.payload as PlayerExecutedPayload | undefined;
        if (!p) return;
        const victim = players[p.playerId];
        setPlayers((prev) => {
          const existing = prev[p.playerId];
          if (!existing) return prev;
          return { ...prev, [p.playerId]: { ...existing, isAlive: false } };
        });
        setExecution({
          playerName: victim?.displayName ?? "Delegate",
          countryName: victim?.countryName ?? "",
          wasRogue: !!p.wasRogue,
          at: Date.now(),
        });
        // Auto-fire the nuclear-strike narrator cue so the Committee
        // voices each strike as it lands. Keyed on eventId so React
        // StrictMode double-invokes don't double-fire the LLM.
        // Roles are scrubbed on the public snapshot mid-game, so we
        // can only speak to the Prime flag here. Saying "HUMAN" for
        // every non-Prime strike would leak info if the target was
        // actually a Replicant; "not the Prime Replicant" preserves
        // the mystery while still giving the narrator something to
        // voice. The post-game roster reveals the real identities.
        fireCue(`execution:${env.event.eventId}`, "execution", {
          country: victim?.countryName ?? "the unidentified delegation",
          name: victim?.displayName ?? "the delegate",
          trueIdentity: p.wasRogue ? "the Prime Replicant" : "not the Prime Replicant",
        });
      }

      if (type === "game_ended") {
        const p = env.payload as { winner?: Party; winCondition?: WinCondition } | undefined;
        if (p?.winner) setFinalWinner(p.winner);
        if (p?.winCondition) setFinalCondition(p.winCondition);
        setGame((prev) => (prev ? { ...prev, status: "completed", phase: "game_over" } : prev));
        if (p?.winner && p?.winCondition) {
          fireCue("closing", "closing", {
            winner: winnerLabel(p.winner),
            condition: humanCondition(p.winCondition),
          });
        }
        // Roster refetch: the backend stops scrubbing roles once the
        // game is completed, so this re-hydrates `players` with true
        // identities for the post-mortem.
        const g = game;
        const tok = token;
        if (g && tok) {
          (async () => {
            try {
              const api = new HostApi({ token: tok });
              const snap = await api.getGame(g.gameId);
              const ix: Record<string, Player> = {};
              for (const pl of snap.players) ix[pl.playerId] = pl;
              setPlayers(ix);
            } catch {
              /* ignore — winner banner still renders without roles */
            }
          })();
        }
        clearSession();
      }

      if (type === "narrator_speak") {
        const p = env.payload as NarratorSpeakPayload | undefined;
        if (!p) return;
        setNarratorCue(p);
      }

      if (type === "cable_phase_opened") {
        // A new Cable Phase starts — clear any stale leak from the
        // previous round so the host TV is clean for the next one.
        setCableLeak(null);
      }

      if (type === "cable_leaked") {
        const p = env.payload as CableLeakedPayload | undefined;
        if (!p) return;
        setCableLeak(p);
      }
    },
  });

  // Dismiss TERMINATED overlay after 4s.
  const execTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (!execution) return;
    if (execTimerRef.current) clearTimeout(execTimerRef.current);
    execTimerRef.current = setTimeout(() => setExecution(null), 4000);
    return () => {
      if (execTimerRef.current) clearTimeout(execTimerRef.current);
    };
  }, [execution]);

  const seated = useMemo(
    () => Object.values(players).sort((a, b) => a.seat - b.seat),
    [players],
  );
  const presName = presidentPlayerId ? players[presidentPlayerId]?.displayName ?? "" : "";
  const chanName = chancellorPlayerId ? players[chancellorPlayerId]?.displayName ?? "" : "";

  return (
    <div style={{ width: "100vw", height: "100vh" }}>
      <RPTVChrome
        title={
          game?.status === "completed"
            ? "FINAL DISPOSITION"
            : game
            ? phaseTitle(game.phase)
            : "SESSION"
        }
        nodeId={`NODE ${code.slice(0, 6)}`}
        phase={game ? yearForRound(game.round) : "…"}
      >
        {err && (
          <div
            style={{
              position: "absolute",
              top: 12,
              right: 12,
              border: "1px solid var(--stamp-red)",
              color: "var(--stamp-red)",
              background: "rgba(179,39,36,0.08)",
              padding: "6px 12px",
              fontFamily: "var(--font-mono)",
              fontSize: 11,
              zIndex: 30,
            }}
          >
            {err}
          </div>
        )}

        {game?.status === "completed" ? (
          <WinnerPanel winner={finalWinner} condition={finalCondition} seated={seated} />
        ) : (
          <InGameBoard
            game={game}
            seated={seated}
            presName={presName}
            chanName={chanName}
            presidentPlayerId={presidentPlayerId}
            chancellorPlayerId={chancellorPlayerId}
            votedCount={Object.keys(votedSet).length}
            aliveCount={seated.filter((p) => p.isAlive).length}
            lastElection={lastElection}
          />
        )}

        {execution && <TerminatedOverlay ex={execution} />}

        {cableLeak && <CableLeakOverlay payload={cableLeak} />}

        {narratorCue && (
          <HostNarrator
            payload={narratorCue}
            onComplete={() => setNarratorCue(null)}
          />
        )}

        {/* Host-only SPEAK control. Hidden while narrator is busy or
            the game is over. */}
        {token && game && game.status !== "completed" && !narratorCue && (
          <button
            disabled={narratorBusy}
            onClick={async () => {
              if (!game) return;
              setNarratorBusy(true);
              try {
                const api = new HostApi({ token });
                await api.narrate(game.gameId, pickAutoCue(game));
              } catch (e) {
                setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
              } finally {
                setNarratorBusy(false);
              }
            }}
            style={{
              position: "absolute",
              right: 18,
              bottom: 56,
              zIndex: 30,
              background: rpColors.cyan,
              color: rpColors.broadcast,
              border: `2px solid ${rpColors.cyan}`,
              padding: "10px 16px",
              fontFamily: "var(--font-display)",
              fontSize: 13,
              fontWeight: 700,
              letterSpacing: "0.14em",
              boxShadow: "3px 3px 0 rgba(0,0,0,0.5)",
              cursor: narratorBusy ? "wait" : "pointer",
              opacity: narratorBusy ? 0.5 : 1,
            }}
          >
            {narratorBusy ? "◼ SYNTHESIZING…" : "▸ SPEAK"}
          </button>
        )}
      </RPTVChrome>
    </div>
  );
}

/** Pick a sensible cue based on current phase — used by the SPEAK
 *  button when no event-specific cue has been requested. */
function pickAutoCue(g: Game): string {
  switch (g.phase) {
    case "nomination":
      return "opening";
    case "game_over":
      return "closing";
    default:
      return "opening";
  }
}

// ─── In-game dispatcher ───────────────────────────────────────────

function InGameBoard({
  game,
  seated,
  presName,
  chanName,
  presidentPlayerId,
  chancellorPlayerId,
  votedCount,
  aliveCount,
  lastElection,
}: {
  game: Game | null;
  seated: Player[];
  presName: string;
  chanName: string;
  presidentPlayerId: string | null;
  chancellorPlayerId: string | null;
  votedCount: number;
  aliveCount: number;
  lastElection: LastElection | null;
}) {
  return (
    <div
      style={{
        height: "100%",
        padding: "40px 72px",
        display: "grid",
        gridTemplateColumns: "1.2fr 1fr",
        gap: 56,
      }}
    >
      {/* LEFT — phase-specific stage */}
      <div style={{ display: "flex", flexDirection: "column", gap: 32, minHeight: 0 }}>
        <PhaseStage
          game={game}
          presName={presName}
          chanName={chanName}
          votedCount={votedCount}
          aliveCount={aliveCount}
          lastElection={lastElection}
        />
      </div>

      {/* RIGHT — policy track + assembly */}
      <div style={{ display: "flex", flexDirection: "column", gap: 24, minHeight: 0 }}>
        <PolicyTrack game={game} />
        <Assembly
          seated={seated}
          presidentPlayerId={presidentPlayerId}
          chancellorPlayerId={chancellorPlayerId}
        />
      </div>
    </div>
  );
}

function PhaseStage({
  game,
  presName,
  chanName,
  votedCount,
  aliveCount,
  lastElection,
}: {
  game: Game | null;
  presName: string;
  chanName: string;
  votedCount: number;
  aliveCount: number;
  lastElection: LastElection | null;
}) {
  if (!game) return <BigLabel eyebrow="LOADING" title="RECEIVING" />;
  const phase = game.phase;

  const content = (() => {
    switch (phase) {
      case "nomination":
        return (
          <BigLabel
            eyebrow="NOMINATION"
            title={"AWAITING\nCANDIDATE"}
            memo={
              presName
                ? `"${titleCase(presName)}, President of this Committee, is selecting an Envoy."`
                : '"The Committee is drafting a nomination."'
            }
          />
        );
      case "cable_phase":
        return (
          <BigLabel
            eyebrow="CABLE PHASE"
            title={"ENCRYPTED\nTRAFFIC"}
            memo={
              presName && chanName
                ? `"${titleCase(presName)} has tabled ${titleCase(chanName)} as their Envoy. Diplomatic cables are open. The Committee is listening."`
                : '"Diplomatic cables are open. The Committee is listening."'
            }
          />
        );
      case "election":
        return (
          <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
            <BigLabel
              eyebrow="TRIBUNAL · ELECTION"
              title={"CAST YOUR\nVERDICT"}
              memo={
                presName && chanName
                  ? `"${titleCase(presName)} presents ${titleCase(chanName)} as Envoy. The Committee is voting."`
                  : '"The Committee is voting."'
              }
            />
            <VoteMeter
              votedCount={votedCount}
              aliveCount={aliveCount}
              presName={presName}
              chanName={chanName}
            />
          </div>
        );
      case "legislative_president":
        return (
          <BigLabel
            eyebrow="SORTING ROOM"
            title={"PRESIDENT\nREVIEWING"}
            memo={`"${titleCase(presName)} is reviewing three protocols."`}
          />
        );
      case "legislative_chancellor":
        return (
          <BigLabel
            eyebrow="DRAFTING FLOOR"
            title={"ENVOY\nDRAFTING"}
            memo={`"${titleCase(chanName)} is selecting a protocol to enact."`}
          />
        );
      case "veto_requested":
        return (
          <BigLabel
            eyebrow="VETO CONSIDERATION"
            title={"VETO\nPROPOSED"}
            memo={`"${titleCase(chanName)} has moved to strike the agenda. ${titleCase(presName)} must concur."`}
          />
        );
      case "executive_action":
        return (
          <BigLabel
            eyebrow="EXECUTIVE ORDER"
            title={execPhaseTitle(game.pendingActionType)}
            memo={`"${titleCase(presName)} is exercising an executive power."`}
          />
        );
      case "lobby":
        return <BigLabel eyebrow="INTAKE" title={"AWAITING\nDEPLOYMENT"} />;
      case "game_over":
        return <BigLabel eyebrow="FINAL" title={"SESSION\nCOMPLETE"} />;
    }
  })();

  return (
    <>
      {content}
      {lastElection && <LastElectionBanner e={lastElection} />}
    </>
  );
}

function BigLabel({
  eyebrow,
  title,
  memo,
}: {
  eyebrow: string;
  title: string;
  memo?: string;
}) {
  return (
    <div>
      <div className="t-eyebrow" style={{ color: rpColors.cyan, marginBottom: 14 }}>
        ▸ {eyebrow}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 128,
          lineHeight: 0.9,
          color: rpColors.paper3,
          letterSpacing: "-0.02em",
          textTransform: "uppercase",
          position: "relative",
        }}
      >
        <span
          aria-hidden="true"
          style={{ position: "absolute", left: 6, top: 6, color: rpColors.stampRed, opacity: 0.35 }}
        >
          {title}
        </span>
        <span style={{ position: "relative", whiteSpace: "pre-line" }}>{title}</span>
      </div>
      {memo && (
        <div
          style={{
            marginTop: 16,
            fontFamily: "var(--font-typewriter)",
            fontSize: 20,
            color: rpColors.paper3,
            opacity: 0.75,
            maxWidth: 720,
          }}
        >
          {memo}
        </div>
      )}
    </div>
  );
}

function VoteMeter({
  votedCount,
  aliveCount,
  presName,
  chanName,
}: {
  votedCount: number;
  aliveCount: number;
  presName: string;
  chanName: string;
}) {
  const pct = aliveCount > 0 ? (votedCount / aliveCount) * 100 : 0;
  return (
    <div
      style={{
        border: `1px solid ${rpColors.broadcastRule}`,
        padding: "18px 22px",
        background: "rgba(95,211,204,0.04)",
      }}
    >
      <div className="t-eyebrow" style={{ color: rpColors.cyan, marginBottom: 10 }}>
        BALLOTS CAST
      </div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          marginBottom: 10,
        }}
      >
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 56,
            color: rpColors.paper3,
            lineHeight: 0.9,
          }}
        >
          {String(votedCount).padStart(2, "0")}
          <span style={{ color: rpColors.inkFaded, fontSize: 30 }}>
            /{String(aliveCount).padStart(2, "0")}
          </span>
        </div>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 13,
            color: rpColors.inkFaded,
            lineHeight: 1.6,
            textAlign: "right",
          }}
        >
          PRES · {titleCase(presName).toUpperCase() || "—"}
          <br />
          ENVOY · {titleCase(chanName).toUpperCase() || "—"}
        </div>
      </div>
      <div
        style={{
          height: 10,
          background: rpColors.broadcast2,
          border: `1px solid ${rpColors.broadcastRule}`,
          position: "relative",
        }}
      >
        <div
          style={{
            height: "100%",
            width: `${pct}%`,
            background: rpColors.cyan,
            transition: "width 0.3s ease",
          }}
        />
      </div>
    </div>
  );
}

function LastElectionBanner({ e }: { e: LastElection }) {
  const accent = e.passed ? rpColors.stampGreen : rpColors.stampRed;
  return (
    <div
      style={{
        border: `1px solid ${accent}`,
        background: "rgba(255,255,255,0.02)",
        padding: "12px 16px",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 20,
        fontFamily: "var(--font-mono)",
        fontSize: 13,
        color: rpColors.paper3,
        letterSpacing: 1.2,
      }}
    >
      <span className="t-eyebrow" style={{ color: accent, fontSize: 11 }}>
        LAST TRIBUNAL
      </span>
      <span>
        {titleCase(e.presidentName).toUpperCase()} + {titleCase(e.chancellorName).toUpperCase()}
      </span>
      <span>
        YEA <b>{String(e.jaVotes).padStart(2, "0")}</b> · NAY <b>{String(e.neinVotes).padStart(2, "0")}</b>
      </span>
      <span
        style={{
          color: accent,
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          letterSpacing: "0.14em",
        }}
      >
        {e.passed ? "PASSED" : "REJECTED"}
      </span>
    </div>
  );
}

function PolicyTrack({ game }: { game: Game | null }) {
  const human = game?.humanPoliciesEnacted ?? 0;
  const ai = game?.aiPoliciesEnacted ?? 0;
  const tracker = game?.electionTracker ?? 0;
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div className="t-eyebrow" style={{ color: rpColors.cyan }}>
        PROTOCOL BOARD
      </div>
      <div style={{ display: "flex", gap: 12 }}>
        <PolicyBar label="HUMAN" count={human} slots={5} color={rpColors.cyan} />
        <PolicyBar label="AI" count={ai} slots={6} color={rpColors.stampRed} />
      </div>
      <div
        style={{
          border: `1px solid ${rpColors.broadcastRule}`,
          padding: "8px 12px",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          fontFamily: "var(--font-mono)",
          fontSize: 11,
          color: rpColors.inkFaded,
          letterSpacing: 1.4,
        }}
      >
        <span>ELECTION TRACKER</span>
        <span style={{ display: "flex", gap: 4 }}>
          {[0, 1, 2].map((i) => (
            <span
              key={i}
              style={{
                width: 12,
                height: 12,
                background: i < tracker ? rpColors.stampRed : "transparent",
                border: `1px solid ${i < tracker ? rpColors.stampRed : rpColors.broadcastRule}`,
              }}
            />
          ))}
        </span>
        <span>VETO · {game?.vetoUnlocked ? "UNLOCKED" : "LOCKED"}</span>
      </div>
    </div>
  );
}

function PolicyBar({
  label,
  count,
  slots,
  color,
}: {
  label: string;
  count: number;
  slots: number;
  color: string;
}) {
  return (
    <div
      style={{
        flex: 1,
        border: `1px solid ${color}`,
        padding: "10px 12px",
        background: "rgba(255,255,255,0.02)",
      }}
    >
      <div
        style={{
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          color,
          letterSpacing: 1.4,
          marginBottom: 6,
        }}
      >
        {label} · {String(count).padStart(2, "0")}/{String(slots).padStart(2, "0")}
      </div>
      <div style={{ display: "flex", gap: 4 }}>
        {Array.from({ length: slots }).map((_, i) => (
          <div
            key={i}
            style={{
              flex: 1,
              height: 22,
              background: i < count ? color : "transparent",
              border: `1px solid ${color}`,
            }}
          />
        ))}
      </div>
    </div>
  );
}

function Assembly({
  seated,
  presidentPlayerId,
  chancellorPlayerId,
}: {
  seated: Player[];
  presidentPlayerId: string | null;
  chancellorPlayerId: string | null;
}) {
  return (
    <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: 12, minHeight: 0 }}>
      <div className="t-eyebrow" style={{ color: rpColors.cyan }}>
        ASSEMBLY · {seated.filter((p) => p.isAlive).length}/
        {String(seated.length).padStart(2, "0")} ACTIVE
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: 8,
          overflow: "auto",
          minHeight: 0,
        }}
      >
        {seated.map((p) => {
          const isPres = p.playerId === presidentPlayerId;
          const isChan = p.playerId === chancellorPlayerId;
          const dead = !p.isAlive;
          return (
            <div
              key={p.playerId}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                border: `1px solid ${dead ? rpColors.stampRed : rpColors.broadcastRule}`,
                padding: "8px 12px",
                background: dead ? "rgba(179,39,36,0.06)" : "rgba(255,255,255,0.02)",
                opacity: dead ? 0.6 : 1,
              }}
            >
              <div
                style={{
                  width: 36,
                  height: 36,
                  background: dead ? rpColors.stampRed : rpColors.broadcastRule,
                  color: rpColors.paper3,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontFamily: "var(--font-display)",
                  fontWeight: 700,
                  fontSize: 18,
                }}
              >
                {p.displayName.trim().slice(0, 1).toUpperCase() || "?"}
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontFamily: "var(--font-mono)",
                    fontSize: 10,
                    color: rpColors.inkFaded,
                    letterSpacing: 1.3,
                  }}
                >
                  SEAT {String(p.seat + 1).padStart(2, "0")}
                  {p.countryCode && ` · ${p.countryCode}`}
                </div>
                <div
                  style={{
                    fontFamily: "var(--font-display)",
                    fontWeight: 600,
                    fontSize: 16,
                    color: rpColors.paper3,
                    letterSpacing: "0.02em",
                    textTransform: "uppercase",
                  }}
                >
                  {p.displayName}
                </div>
                {p.countryName && (
                  <div
                    style={{
                      fontFamily: "var(--font-mono)",
                      fontSize: 9,
                      color: rpColors.cyanSoft,
                      letterSpacing: 1.2,
                      textTransform: "uppercase",
                      marginTop: 1,
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {p.countryName}
                  </div>
                )}
              </div>
              {isPres && <Badge color={rpColors.cyan}>P</Badge>}
              {isChan && <Badge color={rpColors.stampRed}>C</Badge>}
              {dead && <Badge color={rpColors.stampRed}>✖</Badge>}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function Badge({ color, children }: { color: string; children: React.ReactNode }) {
  return (
    <span
      style={{
        border: `1px solid ${color}`,
        color,
        fontFamily: "var(--font-display)",
        fontWeight: 700,
        fontSize: 12,
        letterSpacing: "0.1em",
        padding: "2px 6px",
      }}
    >
      {children}
    </span>
  );
}

// ─── NightScreen ─────────────────────────────────────────────────
//
// Lights-out treatment for the two phases where every phone but the
// President's (and briefly the Chancellor's) is idle. The TV goes
// dramatic: giant CHAMBER word, 3-faction instruction grid, live
// countdown bound to `game.phaseDeadline` when present.

function isChamberQuiet(phase: Game["phase"]): boolean {
  return phase === "legislative_president" || phase === "legislative_chancellor";
}

function NightScreen({
  game,
  presName,
  chanName,
}: {
  game: Game;
  presName: string;
  chanName: string;
}) {
  const pres = titleCase(presName).toUpperCase() || "—";
  const chan = titleCase(chanName).toUpperCase() || "—";
  const isPresStage = game.phase === "legislative_president";
  const word = isPresStage ? "SORTING" : "DRAFT";
  const eyebrow = isPresStage ? "▼ THE SORTING ROOM IS SEALED ▼" : "▼ THE DRAFTING FLOOR IS SEALED ▼";
  const subjectMemo = isPresStage
    ? `"${pres} is reviewing three protocols. The chamber observes in silence."`
    : `"${chan} is selecting a protocol to enact. The chamber observes in silence."`;

  const factions: Array<{ eyb: string; txt: string; c: string }> = [
    {
      eyb: "HUMANS",
      txt: isPresStage
        ? "Observe the proceedings.\nDo not speak of them\nafterward."
        : "The verdict is imminent.\nRead the chamber.",
      c: rpColors.cyan,
    },
    {
      eyb: "AI CABAL",
      txt: isPresStage
        ? "Monitor your operative.\nDo not interfere."
        : "Your protocol is on the floor.\nKeep your counsel.",
      c: rpColors.stampRed,
    },
    {
      eyb: "PRIME",
      txt: isPresStage
        ? "Maintain cover.\nThe Committee is watching."
        : "Compose your expression.\nNeutral face.",
      c: rpColors.amber,
    },
  ];

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 36,
        padding: 60,
        animation: "paperSlide 0.35s ease-out both",
      }}
    >
      <div
        className="t-eyebrow"
        style={{ color: rpColors.stampRed, letterSpacing: "0.3em", fontSize: 14 }}
      >
        {eyebrow}
      </div>

      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 240,
          letterSpacing: "-0.02em",
          lineHeight: 0.9,
          color: rpColors.paper3,
          position: "relative",
          textTransform: "uppercase",
        }}
      >
        <span
          aria-hidden="true"
          style={{ position: "absolute", left: 12, top: 12, color: rpColors.stampRed, opacity: 0.5 }}
        >
          {word}
        </span>
        <span style={{ position: "relative" }}>{word}</span>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 320px)", gap: 24 }}>
        {factions.map((card) => (
          <div
            key={card.eyb}
            style={{
              border: `1px solid ${card.c}`,
              padding: 24,
              background: "rgba(255,255,255,0.02)",
              minHeight: 140,
            }}
          >
            <div className="t-eyebrow" style={{ color: card.c, marginBottom: 12 }}>
              ◼ {card.eyb}
            </div>
            <div
              style={{
                fontFamily: "var(--font-typewriter)",
                fontSize: 18,
                color: rpColors.paper3,
                lineHeight: 1.5,
                whiteSpace: "pre-line",
              }}
            >
              {card.txt}
            </div>
          </div>
        ))}
      </div>

      <div
        style={{
          marginTop: 8,
          display: "flex",
          alignItems: "center",
          gap: 40,
          maxWidth: 1000,
        }}
      >
        <DeadlineTimer deadline={game.phaseDeadline ?? null} label="CHAMBER CLOSES" />
        <div
          style={{
            fontFamily: "var(--font-typewriter)",
            fontSize: 17,
            color: rpColors.paper3,
            opacity: 0.75,
            maxWidth: 560,
          }}
        >
          {subjectMemo}
        </div>
      </div>
    </div>
  );
}

/** Renders an RPTimer whose value ticks down from an ISO deadline.
 *  When the deadline is absent we just render `--:--`. */
function DeadlineTimer({
  deadline,
  label,
}: {
  deadline: string | null;
  label: string;
}) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!deadline) return;
    const id = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(id);
  }, [deadline]);

  let display = "--:--";
  let danger = false;
  if (deadline) {
    const deadlineMs = Date.parse(deadline);
    const remaining = Math.max(0, Math.floor((deadlineMs - now) / 1000));
    const m = Math.floor(remaining / 60);
    const s = remaining % 60;
    display = `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
    danger = remaining <= 10 && remaining > 0;
  }

  return <RPTimer value={display} label={label} danger={danger} />;
}

// ─── Winner banner ────────────────────────────────────────────────

function WinnerPanel({
  winner,
  condition,
  seated,
}: {
  winner: Party | null;
  condition: WinCondition | null;
  seated: Player[];
}) {
  const { name: winnerName, verb: winnerVerb, accent } = winnerHeadline(winner);
  const alive = seated.filter((p) => p.isAlive).length;
  const rosterHasRoles = seated.some((p) => p.role);

  return (
    <div
      style={{
        height: "100%",
        padding: "36px 64px",
        display: "grid",
        gridTemplateColumns: "1.2fr 1fr",
        gap: 48,
        animation: "paperSlide 0.5s ease-out both",
      }}
    >
      {/* LEFT — headline */}
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          gap: 20,
        }}
      >
        <div className="t-eyebrow" style={{ color: accent }}>
          ◼ FINAL DISPOSITION
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 172,
            letterSpacing: "-0.03em",
            lineHeight: 0.88,
            color: accent,
            position: "relative",
          }}
        >
          <span
            aria-hidden="true"
            style={{ position: "absolute", left: 8, top: 8, color: rpColors.stampRed, opacity: 0.35 }}
          >
            {winnerName}
          </span>
          <span style={{ position: "relative" }}>{winnerName}</span>
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 80,
            color: rpColors.paper3,
            letterSpacing: "-0.02em",
            lineHeight: 0.9,
          }}
        >
          {winnerVerb}
        </div>
        <div
          style={{
            fontFamily: "var(--font-typewriter)",
            fontSize: 20,
            color: rpColors.paper3,
            opacity: 0.75,
            maxWidth: 680,
          }}
        >
          {condition ? winMemo(condition) : '"The Committee has filed its findings."'}
        </div>
        <div
          style={{
            marginTop: 8,
            fontFamily: "var(--font-mono)",
            fontSize: 13,
            color: rpColors.inkFaded,
            letterSpacing: 1.4,
          }}
        >
          POPULATION STATUS · {alive} ACTIVE · {seated.length - alive} STRUCK
        </div>
      </div>

      {/* RIGHT — roster with true identities */}
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 14,
          minHeight: 0,
          justifyContent: "center",
        }}
      >
        <div className="t-eyebrow" style={{ color: rpColors.cyan }}>
          ◼ DECLASSIFIED ROSTER · TRUE IDENTITIES
        </div>
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 6,
            overflow: "auto",
            minHeight: 0,
          }}
        >
          {seated.map((p) => (
            <RosterRow key={p.playerId} player={p} />
          ))}
        </div>
        {!rosterHasRoles && (
          <div
            style={{
              fontFamily: "var(--font-mono)",
              fontSize: 11,
              color: rpColors.inkFaded,
              letterSpacing: 1.2,
            }}
          >
            · awaiting declassified manifest ·
          </div>
        )}
      </div>
    </div>
  );
}

function RosterRow({ player }: { player: Player }) {
  const dead = !player.isAlive;
  const r = player.role as Role | "" | undefined;
  const { label, color } = roleBadge(r);
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 14,
        border: `1px solid ${dead ? rpColors.stampRed : rpColors.broadcastRule}`,
        background: dead ? "rgba(179,39,36,0.06)" : "rgba(255,255,255,0.02)",
        padding: "10px 14px",
        opacity: dead ? 0.72 : 1,
      }}
    >
      <div
        style={{
          width: 42,
          height: 42,
          background: dead ? rpColors.stampRed : color,
          color: rpColors.broadcast,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 20,
        }}
      >
        {player.displayName.trim().slice(0, 1).toUpperCase() || "?"}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            color: rpColors.inkFaded,
            letterSpacing: 1.3,
          }}
        >
          SEAT {String(player.seat + 1).padStart(2, "0")}
          {player.countryCode && ` · ${player.countryCode}`}
          {player.countryName && ` · ${player.countryName.toUpperCase()}`}
          {dead && " · STRUCK"}
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 600,
            fontSize: 18,
            color: rpColors.paper3,
            letterSpacing: "0.02em",
            textTransform: "uppercase",
          }}
        >
          {player.displayName}
        </div>
      </div>
      <span
        style={{
          border: `1px solid ${color}`,
          color,
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 12,
          letterSpacing: "0.14em",
          padding: "4px 10px",
          whiteSpace: "nowrap",
        }}
      >
        {label}
      </span>
    </div>
  );
}

/** Plain-English label for a winning Party — used in narrator vars
 *  so the closing cue reads naturally. */
function winnerLabel(p: Party): string {
  switch (p) {
    case "human":
      return "the Humans";
    case "ai":
      return "the Replicants";
    case "singularity":
      return "the Singularity";
  }
}

/** Maps the winning Party to the big-text headline (name + verb) and
 *  accent color. Name renders in accent; verb renders in paper-white
 *  at half the size. Singularity gets a distinct "ASCENDS" treatment
 *  since it won alone, not as a faction bloc. */
function winnerHeadline(winner: Party | null): {
  name: string;
  verb: string;
  accent: string;
} {
  switch (winner) {
    case "human":
      return { name: "HUMANS", verb: "PREVAIL", accent: rpColors.cyan };
    case "ai":
      return { name: "REPLICANTS", verb: "PREVAIL", accent: rpColors.stampRed };
    case "singularity":
      return { name: "SINGULARITY", verb: "ASCENDS", accent: rpColors.amber };
    default:
      return { name: "NO VERDICT", verb: "FILED", accent: rpColors.inkFaded };
  }
}

function roleBadge(r: Role | "" | undefined): { label: string; color: string } {
  switch (r) {
    case "human":
      return { label: "HUMAN", color: rpColors.cyan };
    case "ai":
      return { label: "AI CABAL", color: rpColors.stampRed };
    case "rogue":
      return { label: "PRIME", color: rpColors.amber };
    case "singularity":
      return { label: "SINGULARITY", color: rpColors.amber };
    default:
      return { label: "REDACTED", color: rpColors.inkFaded };
  }
}

function winMemo(c: WinCondition): string {
  switch (c) {
    case "human_policies":
      return '"Five human policies ratified. The meltdown has been held back; humanity endures."';
    case "ai_policies":
      return '"Six AI policies passed. The reactors go critical; the collapse unfolds."';
    case "rogue_elected_chancellor":
      return '"The Prime has been seated as Envoy with the codes live. The Committee yields its keys."';
    case "rogue_executed":
      return '"The Prime\'s nation has been struck from the register. The Committee resumes normal proceedings."';
    case "singularity_kingmaker":
      return '"The codes have passed to the Singularity. The Committee adjourns without a faction victor — a solitary actor has claimed the endgame."';
  }
}

/** Plain-language version of a WinCondition — passed to the narrator
 *  LLM as a var so its closing line reads naturally. */
function humanCondition(c: WinCondition): string {
  switch (c) {
    case "human_policies":
      return "five Human protocols were ratified";
    case "ai_policies":
      return "six AI protocols were passed";
    case "rogue_elected_chancellor":
      return "the Prime was seated as Envoy while the codes were live";
    case "rogue_executed":
      return "the Prime's nation was struck by retaliatory order";
    case "singularity_kingmaker":
      return "the Singularity took the Envoy's seat while the codes were live";
  }
}

// ─── TERMINATED overlay ───────────────────────────────────────────

// ─── Cable leak overlay ───────────────────────────────────────────
//
// Rendered when a cable_leaked envelope arrives. Silenced leaks show
// a muted "NO TRAFFIC FLAGGED" card; flagged leaks render the author
// + quoted body with a big red CLASSIFIED/INTERCEPTED stamp. The
// overlay stays on screen through the election vote so players can
// reference it — clears on the next cable_phase_opened.

function CableLeakOverlay({ payload }: { payload: CableLeakedPayload }) {
  if (payload.silenced) {
    return (
      <div
        style={{
          position: "absolute",
          left: 40,
          top: 40,
          zIndex: 22,
          maxWidth: 520,
          padding: "14px 18px",
          border: `1px solid ${rpColors.broadcastRule}`,
          background: "rgba(20, 22, 26, 0.88)",
          animation: "paperSlide 0.3s ease-out both",
        }}
      >
        <div
          className="t-eyebrow"
          style={{ color: rpColors.inkFaded, marginBottom: 6, fontSize: 11 }}
        >
          ◼ CABLE REVIEW · NO TRAFFIC FLAGGED
        </div>
        <div
          style={{
            fontFamily: "var(--font-typewriter)",
            fontSize: 14,
            color: rpColors.paper3,
            opacity: 0.75,
            lineHeight: 1.4,
          }}
        >
          &quot;The Committee reviewed this round&apos;s diplomatic traffic.
          No items warrant broadcast.&quot;
        </div>
      </div>
    );
  }
  return (
    <div
      style={{
        position: "absolute",
        left: 40,
        top: 40,
        zIndex: 22,
        maxWidth: 640,
        padding: "20px 24px",
        border: `2px solid ${rpColors.stampRed}`,
        background: "rgba(20, 22, 26, 0.92)",
        boxShadow: "4px 4px 0 rgba(0,0,0,0.6)",
        animation: "paperSlide 0.3s ease-out both",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 10,
        }}
      >
        <div className="t-eyebrow" style={{ color: rpColors.stampRed, fontSize: 12 }}>
          ◼ INTERCEPTED CABLE · COMMITTEE FLAGGED
        </div>
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            color: rpColors.inkFaded,
            letterSpacing: 1.3,
          }}
        >
          SUBVERSION · {String(Math.round(payload.subversionScore ?? 0)).padStart(2, "0")}/10
        </div>
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 32,
          color: rpColors.paper3,
          letterSpacing: "0.02em",
          textTransform: "uppercase",
          marginBottom: 8,
        }}
      >
        {titleCase(payload.author ?? "—").toUpperCase()}
      </div>
      <div
        style={{
          fontFamily: "var(--font-typewriter)",
          fontSize: 18,
          color: rpColors.paper3,
          lineHeight: 1.45,
          borderLeft: `3px solid ${rpColors.stampRed}`,
          paddingLeft: 14,
          maxWidth: 560,
        }}
      >
        &ldquo;{payload.body}&rdquo;
      </div>
    </div>
  );
}

function TerminatedOverlay({ ex }: { ex: Execution }) {
  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        background: "rgba(6, 8, 10, 0.68)",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 24,
        zIndex: 20,
        animation: "paperSlide 0.3s ease-out both",
      }}
    >
      <div className="t-eyebrow" style={{ color: rpColors.stampRed, fontSize: 14 }}>
        ◼ NUCLEAR STRIKE · RETALIATORY ORDER
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 160,
          letterSpacing: "-0.02em",
          lineHeight: 0.88,
          color: rpColors.stampRed,
          textTransform: "uppercase",
        }}
      >
        {ex.countryName || ex.playerName}
      </div>
      <div
        style={{ fontFamily: "var(--font-typewriter)", fontSize: 22, color: rpColors.paper3, opacity: 0.85 }}
      >
        &ldquo;…has been struck from the register. The delegation
        is vaporised; the seat is void.&rdquo;
      </div>
      <div style={{ marginTop: 20 }}>
        <RPStamp variant="red" rotate={-8} size={56} animate>
          STRUCK
        </RPStamp>
      </div>
      {ex.wasRogue && (
        <div
          style={{
            marginTop: 14,
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 28,
            letterSpacing: "0.12em",
            color: rpColors.amber,
          }}
        >
          TRUE IDENTITY · PRIME REPLICANT
        </div>
      )}
    </div>
  );
}

// ─── helpers ──────────────────────────────────────────────────────

/** Convert an engine round counter (1-indexed) to the narrative year
 *  shown on the host chrome. The first round is 5 years from today —
 *  the Committee convenes in the near future to head off the meltdown. */
function yearForRound(round: number): string {
  if (round <= 0) return "—";
  const base = new Date().getUTCFullYear() + 5;
  return `YEAR ${base + round - 1}`;
}

function phaseTitle(p: Game["phase"]): string {
  switch (p) {
    case "lobby":
      return "INTAKE";
    case "nomination":
      return "NOMINATION";
    case "cable_phase":
      return "CABLE PHASE · ENCRYPTED TRAFFIC";
    case "election":
      return "TRIBUNAL · VOTE";
    case "legislative_president":
      return "SORTING ROOM";
    case "legislative_chancellor":
      return "DRAFTING FLOOR";
    case "veto_requested":
      return "VETO CONSIDERATION";
    case "executive_action":
      return "EXECUTIVE ORDER";
    case "game_over":
      return "FINAL DISPOSITION";
  }
}

function execPhaseTitle(a: Game["pendingActionType"]): string {
  switch (a) {
    case "investigate_loyalty":
      return "INVESTIGATE\nLOYALTY";
    case "special_election":
      return "SPECIAL\nELECTION";
    case "policy_peek":
      return "POLICY\nPEEK";
    case "execution":
      return "NUCLEAR\nSTRIKE";
    case "top_deck":
      return "TOP DECK\nPROTOCOL";
    default:
      return "EXECUTIVE\nORDER";
  }
}

function titleCase(s: string): string {
  return s
    .split(" ")
    .map((w) => (w[0] ? w[0].toUpperCase() + w.slice(1).toLowerCase() : w))
    .join(" ");
}
