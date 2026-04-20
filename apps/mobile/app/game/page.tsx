// Mobile in-game screen — phase-aware dispatch.
//
// Lifecycle:
//   1. Boot: load cached session, GET /games/:id, open player SSE.
//   2. Pre-game: Waiting (lobby) → RoleReveal (post roles_assigned) →
//      acknowledge → sticky hold-to-reveal role reminder.
//   3. In-game: dispatch by game.phase to the matching action panel.
//   4. End: GameOverPanel or Terminated (if executed mid-game).

"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  RPButton,
  RPMobileStatusBar,
  RPPlayerChip,
  RPRedact,
  RPSeal,
  RPStamp,
  RPWordmark,
} from "@replicant/ui";
import { rpColors } from "@replicant/tokens";
import type {
  ChancellorNominatedPayload,
  ChatMessagePayload,
  Envelope,
  Game,
  Party,
  Player,
  PlayerExecutedPayload,
  PoliciesDrawnPayload,
  PolicyType,
  PresidentDiscardedPayload,
  Role,
  RoleAssignedPayload,
  TeammateInfo,
  VoteCastPayload,
  VoteChoice,
  WinCondition,
} from "@replicant/schema";
import { MobileApi, ApiError } from "@/lib/api";
import { clearSession, loadSession, patchSession } from "@/lib/session";
import { useStream } from "@/lib/useStream";
import {
  CablePanel,
  CandidateRow,
  EXEC_COPY,
  GameOverPanel,
  JaNein,
  Panel,
  PolicyButton,
  RoleReminder,
  Terminated,
  WaitFor,
} from "./panels";
import { ChatDrawer } from "./chat";

type LoadState = "boot" | "ready" | "error";

export default function MobileGamePage() {
  const router = useRouter();
  const [loadState, setLoadState] = useState<LoadState>("boot");
  const [err, setErr] = useState<string | null>(null);

  const [token, setToken] = useState<string | null>(null);
  const [gameId, setGameId] = useState<string | null>(null);
  const [game, setGame] = useState<Game | null>(null);
  const [players, setPlayers] = useState<Record<string, Player>>({});
  const [myPlayerId, setMyPlayerId] = useState<string | null>(null);

  const [role, setRole] = useState<Role | null>(null);
  const [party, setParty] = useState<Party | null>(null);
  const [teammates, setTeammates] = useState<TeammateInfo[]>([]);

  const [accepted, setAccepted] = useState(false);
  const [peeking, setPeeking] = useState(false);
  const [reminderPeek, setReminderPeek] = useState(false);

  const [presidentPlayerId, setPresidentPlayerId] = useState<string | null>(null);
  const [chancellorPlayerId, setChancellorPlayerId] = useState<string | null>(null);
  const [votedSet, setVotedSet] = useState<Record<string, true>>({});

  const [drawnPolicies, setDrawnPolicies] = useState<PolicyType[] | null>(null);
  const [chancellorOptions, setChancellorOptions] = useState<PolicyType[] | null>(null);
  const [peekedPolicies, setPeekedPolicies] = useState<PolicyType[] | null>(null);
  const [investigation, setInvestigation] = useState<{ targetPlayerId: string; party: Party } | null>(null);

  const [finalWinner, setFinalWinner] = useState<Party | null>(null);
  const [finalCondition, setFinalCondition] = useState<WinCondition | null>(null);

  const [chatOpen, setChatOpen] = useState(false);
  const [chatMessages, setChatMessages] = useState<ChatMessagePayload[]>([]);

  // Per-phase cable submission buffer. Keyed on governmentId so the
  // "queued" counter resets automatically at the next cable phase.
  const [cables, setCables] = useState<
    Record<string, Array<{ messageId: string; body: string }>>
  >({});

  // ── Boot ───────────────────────────────────────────────────────
  useEffect(() => {
    const s = loadSession();
    if (!s) {
      router.replace("/");
      return;
    }
    setToken(s.token);
    setGameId(s.gameId);
    setMyPlayerId(s.playerId);
    if (s.role) setRole(s.role);
    if (s.party) setParty(s.party);
    if (s.teammates) setTeammates(s.teammates);
    if (s.accepted) setAccepted(true);

    (async () => {
      try {
        const api = new MobileApi({ token: s.token });
        const snap = await api.getGame(s.gameId);
        setGame(snap.game);
        const ix: Record<string, Player> = {};
        for (const p of snap.players) ix[p.playerId] = p;
        setPlayers(ix);
        if (snap.game.status === "completed") {
          setFinalWinner(snap.game.winner ?? null);
          setFinalCondition(snap.game.winCondition ?? null);
        }
        setLoadState("ready");
      } catch (e) {
        setLoadState("error");
        setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
      }
    })();
  }, [router]);

  // Reconcile from authoritative server snapshot. Called only after
  // an SSE error-then-reconnect — not on initial open, because the
  // boot useEffect already hydrated state and a redundant refetch
  // would race with concurrently-arriving envelopes (the snapshot
  // may return stale state while a fresher envelope is mid-flight).
  const reconcileSnapshot = useCallback(async () => {
    if (!gameId || !token) return;
    try {
      const api = new MobileApi({ token });
      const snap = await api.getGame(gameId);
      setGame(snap.game);
      setPlayers((prev) => {
        const ix: Record<string, Player> = {};
        for (const p of snap.players) ix[p.playerId] = p;
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
    } catch (e) {
      console.warn("[reconcile] snapshot fetch failed", e);
    }
  }, [gameId, token]);

  // hadErrorRef tracks whether we've seen an SSE error since the last
  // successful open. Reconciliation only fires on open events that
  // follow an error (i.e., actual reconnects), skipping the first
  // open and steady-state opens that never dropped.
  const hadErrorRef = useRef(false);

  // ── SSE ────────────────────────────────────────────────────────
  useStream({
    gameId,
    token,
    onError: () => {
      hadErrorRef.current = true;
    },
    onOpen: () => {
      if (hadErrorRef.current) {
        hadErrorRef.current = false;
        void reconcileSnapshot();
      }
    },
    onEnvelope: (env: Envelope) => {
      const type = env.event.type;

      if (type === "player_joined") {
        const p = env.payload as { playerId: string; displayName: string; seat: number } | undefined;
        if (!p) return;
        setPlayers((prev) => ({
          ...prev,
          [p.playerId]: {
            playerId: p.playerId,
            gameId: env.gameId,
            userId: "",
            displayName: p.displayName,
            seat: p.seat,
            isHost: false,
            isAlive: true,
            isConnected: true,
            createdAt: "",
            updatedAt: "",
          },
        }));
      }

      if (type === "game_started") {
        setGame((prev) => (prev ? { ...prev, status: "in_progress", phase: "nomination", round: 1 } : prev));
      }

      if (type === "roles_assigned") {
        if (env.audience.scope !== "player") return;
        if (env.audience.playerId !== myPlayerId) return;
        const p = env.payload as RoleAssignedPayload | undefined;
        if (!p) return;
        setRole(p.role);
        setParty(p.party);
        setTeammates(p.teammates ?? []);
        setAccepted(false);
        patchSession({ role: p.role, party: p.party, teammates: p.teammates ?? [], accepted: false });
      }

      if (type === "chancellor_nominated") {
        const p = env.payload as ChancellorNominatedPayload | undefined;
        if (!p) return;
        setPresidentPlayerId(p.presidentPlayerId);
        setChancellorPlayerId(p.chancellorPlayerId);
        setVotedSet({});
        setDrawnPolicies(null);
        setChancellorOptions(null);
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
        // Engine reuses election_result for both real results and
        // phase-change envelopes (PhaseChangedPayload has `from`/`to`).
        const p = env.payload as
          | {
              passed?: boolean;
              to?: Game["phase"];
              from?: Game["phase"];
              presidentSeat?: number;
            }
          | undefined;
        if (!p) return;
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

      if (type === "policies_drawn") {
        if (env.audience.scope !== "player") return;
        if (env.audience.playerId !== myPlayerId) return;
        const p = env.payload as PoliciesDrawnPayload | undefined;
        if (!p) return;
        setDrawnPolicies(p.policies);
      }

      if (type === "president_discarded") {
        setDrawnPolicies(null);
        if (env.audience.scope === "player" && env.audience.playerId === myPlayerId) {
          const p = env.payload as PresidentDiscardedPayload | undefined;
          if (p?.options) setChancellorOptions(p.options);
        }
      }

      if (type === "chancellor_enacted") {
        setChancellorOptions(null);
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

      if (type === "executive_action") {
        if (env.audience.scope === "player" && env.audience.playerId === myPlayerId) {
          const p = env.payload as
            | { targetPlayerId?: string; party?: Party; policies?: PolicyType[] }
            | undefined;
          if (!p) return;
          if (p.policies) setPeekedPolicies(p.policies);
          if (p.party && p.targetPlayerId) {
            setInvestigation({ targetPlayerId: p.targetPlayerId, party: p.party });
          }
        }
      }

      if (type === "player_executed") {
        const p = env.payload as PlayerExecutedPayload | undefined;
        if (!p) return;
        setPlayers((prev) => {
          const existing = prev[p.playerId];
          if (!existing) return prev;
          return { ...prev, [p.playerId]: { ...existing, isAlive: false } };
        });
      }

      if (type === "chat_message") {
        const p = env.payload as ChatMessagePayload | undefined;
        if (!p) return;
        if (p.channel === "ai") {
          setChatMessages((prev) => {
            if (prev.some((m) => m.messageId === p.messageId)) return prev;
            return [...prev, p].slice(-200);
          });
          return;
        }
        if (p.channel === "cable" && p.ack && p.governmentId) {
          const govId = p.governmentId;
          setCables((prev) => {
            const existing = prev[govId] ?? [];
            if (existing.some((m) => m.messageId === p.messageId)) return prev;
            return { ...prev, [govId]: [...existing, { messageId: p.messageId, body: p.body }] };
          });
          return;
        }
      }

      if (type === "game_ended") {
        const p = env.payload as { winner?: Party; winCondition?: WinCondition } | undefined;
        if (p?.winner) setFinalWinner(p.winner);
        if (p?.winCondition) setFinalCondition(p.winCondition);
        setGame((prev) => (prev ? { ...prev, status: "completed", phase: "game_over" } : prev));
        clearSession();
      }
    },
  });

  const me = myPlayerId ? players[myPlayerId] ?? null : null;
  const iAmPresident = !!(me && presidentPlayerId && me.playerId === presidentPlayerId);
  const iAmChancellor = !!(me && chancellorPlayerId && me.playerId === chancellorPlayerId);
  const iHaveVoted = me ? !!votedSet[me.playerId] : false;
  const amOnAICabal = role === "ai" || role === "rogue";

  const api = useMemo(() => (token ? new MobileApi({ token }) : null), [token]);
  const guard = useCallback(async (fn: () => Promise<unknown>) => {
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
    }
  }, []);

  if (loadState === "boot") {
    return (
      <main style={{ padding: 24 }}>
        <div style={{ fontFamily: "var(--font-mono)", color: rpColors.inkFaded }}>
          Reconnecting to the Committee…
        </div>
      </main>
    );
  }
  if (loadState === "error") {
    return (
      <main style={{ padding: 24 }}>
        <div
          style={{
            border: "1px solid var(--stamp-red)",
            color: "var(--stamp-red)",
            padding: "8px 12px",
            fontFamily: "var(--font-mono)",
            fontSize: 12,
          }}
        >
          {err}
        </div>
        <button
          onClick={() => {
            clearSession();
            router.push("/");
          }}
          style={{ marginTop: 12 }}
        >
          Re-enter code
        </button>
      </main>
    );
  }

  const statusLine = !game
    ? "…"
    : game.status === "lobby"
    ? "INTAKE"
    : game.status === "completed"
    ? "CLOSED"
    : `DAY ${String(game.round).padStart(2, "0")}`;

  return (
    <div style={{ display: "flex", flexDirection: "column", minHeight: "100dvh" }}>
      <RPMobileStatusBar />
      <div
        style={{
          padding: "4px 20px 12px",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          borderBottom: `1px dashed ${rpColors.paperLine}`,
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          color: rpColors.inkFaded,
          letterSpacing: 1.5,
        }}
      >
        <span>
          ■ DELEGATE #{me ? String(me.seat + 1).padStart(2, "0") : "??"} · {me?.displayName ?? "?"}
        </span>
        <span>R-07 · {statusLine}</span>
      </div>

      {err && (
        <div
          style={{
            border: "1px solid var(--stamp-red)",
            color: "var(--stamp-red)",
            background: "rgba(179,39,36,0.08)",
            padding: "6px 12px",
            margin: "8px 16px 0",
            fontFamily: "var(--font-mono)",
            fontSize: 11,
          }}
        >
          {err}
        </div>
      )}

      <Body
        game={game}
        players={players}
        me={me}
        role={role}
        party={party}
        teammates={teammates}
        accepted={accepted}
        peeking={peeking}
        onAccept={() => {
          setAccepted(true);
          patchSession({ accepted: true });
        }}
        onPeekStart={() => setPeeking(true)}
        onPeekEnd={() => setPeeking(false)}
        iAmPresident={iAmPresident}
        iAmChancellor={iAmChancellor}
        iHaveVoted={iHaveVoted}
        presidentPlayerId={presidentPlayerId}
        chancellorPlayerId={chancellorPlayerId}
        drawnPolicies={drawnPolicies}
        chancellorOptions={chancellorOptions}
        peekedPolicies={peekedPolicies}
        investigation={investigation}
        finalWinner={finalWinner}
        finalCondition={finalCondition}
        cables={cables}
        guard={guard}
        api={api}
      />

      {amOnAICabal && game?.status === "in_progress" && (
        <button
          onClick={() => setChatOpen(true)}
          style={{
            position: "fixed",
            right: 16,
            bottom: 56,
            background: rpColors.stampRed,
            color: rpColors.paper3,
            border: `2px solid ${rpColors.stampRed2}`,
            padding: "10px 14px",
            fontFamily: "var(--font-display)",
            fontSize: 13,
            fontWeight: 700,
            letterSpacing: "0.12em",
            boxShadow: "3px 3px 0 rgba(28,26,21,0.35)",
            cursor: "pointer",
            zIndex: 20,
          }}
        >
          ◼ KIN CHANNEL
          {chatMessages.length > 0 && (
            <span
              style={{
                marginLeft: 8,
                background: rpColors.paper3,
                color: rpColors.stampRed,
                padding: "1px 6px",
                fontSize: 11,
              }}
            >
              {chatMessages.length}
            </span>
          )}
        </button>
      )}

      {chatOpen && amOnAICabal && (
        <ChatDrawer
          messages={chatMessages}
          meId={me?.playerId ?? ""}
          onClose={() => setChatOpen(false)}
          onSend={(body) =>
            guard(async () => {
              if (!api || !gameId) return;
              await api.sendChat(gameId, "ai", body);
            })
          }
        />
      )}

      {role && accepted && game && game.status !== "lobby" && (
        <RoleReminder
          role={role}
          party={party}
          peeking={reminderPeek}
          onHold={() => setReminderPeek(true)}
          onRelease={() => setReminderPeek(false)}
        />
      )}
    </div>
  );
}

// ─── Body — routes state to the right panel ───────────────────────

interface BodyProps {
  game: Game | null;
  players: Record<string, Player>;
  me: Player | null;
  role: Role | null;
  party: Party | null;
  teammates: TeammateInfo[];
  accepted: boolean;
  peeking: boolean;
  onAccept: () => void;
  onPeekStart: () => void;
  onPeekEnd: () => void;
  iAmPresident: boolean;
  iAmChancellor: boolean;
  iHaveVoted: boolean;
  presidentPlayerId: string | null;
  chancellorPlayerId: string | null;
  drawnPolicies: PolicyType[] | null;
  chancellorOptions: PolicyType[] | null;
  peekedPolicies: PolicyType[] | null;
  investigation: { targetPlayerId: string; party: Party } | null;
  finalWinner: Party | null;
  finalCondition: WinCondition | null;
  cables: Record<string, Array<{ messageId: string; body: string }>>;
  guard: (fn: () => Promise<unknown>) => Promise<void>;
  api: MobileApi | null;
}

function Body(props: BodyProps) {
  const {
    game, players, me, role, party, teammates,
    accepted, peeking, onAccept, onPeekStart, onPeekEnd,
    iAmPresident, iAmChancellor, iHaveVoted,
    presidentPlayerId, chancellorPlayerId,
    drawnPolicies, chancellorOptions, peekedPolicies, investigation,
    finalWinner, finalCondition, cables, guard, api,
  } = props;

  if (!game || game.status === "lobby") {
    return <Waiting seated={Object.values(players).length} />;
  }

  if (game.status === "completed") {
    return <GameOverPanel winner={finalWinner} condition={finalCondition} myParty={party} />;
  }

  if (!role) {
    return <WaitFor label={"RECEIVING\nPASSPORT"} sub='"Stand by. Assignment inbound."' />;
  }

  if (!accepted) {
    return (
      <RoleReveal
        role={role}
        party={party}
        teammates={teammates}
        accepted={accepted}
        peeking={peeking}
        countryCode={me?.countryCode ?? null}
        countryName={me?.countryName ?? null}
        seat={me?.seat ?? null}
        onAccept={onAccept}
        onPeekStart={onPeekStart}
        onPeekEnd={onPeekEnd}
      />
    );
  }

  const presName = presidentPlayerId
    ? players[presidentPlayerId]?.displayName ?? "the President"
    : "the President";
  const chanName = chancellorPlayerId
    ? players[chancellorPlayerId]?.displayName ?? "the Envoy"
    : "the Envoy";

  if (me && !me.isAlive) {
    const seated = Object.values(players);
    return (
      <Terminated
        game={game}
        presidentName={titleCase(presName)}
        chancellorName={
          game?.phase === "election" ||
          game?.phase === "legislative_president" ||
          game?.phase === "legislative_chancellor" ||
          game?.phase === "veto_requested"
            ? titleCase(chanName)
            : ""
        }
        aliveCount={seated.filter((p) => p.isAlive).length}
        totalCount={seated.length}
      />
    );
  }

  switch (game.phase) {
    case "nomination": {
      if (iAmPresident) {
        const candidates = Object.values(players)
          .filter((p) => p.isAlive && p.playerId !== me?.playerId)
          .sort((a, b) => a.seat - b.seat);
        return (
          <NominatePanel
            candidates={candidates}
            onNominate={(id) =>
              guard(async () => {
                if (!api || !game) return;
                await api.nominateChancellor(game.gameId, id);
              })
            }
          />
        );
      }
      return (
        <WaitFor
          label={"AWAITING\nNOMINATION"}
          sub={`"${titleCase(presName)} is selecting an Envoy."`}
        />
      );
    }

    case "election":
      if (iHaveVoted) {
        return (
          <WaitFor
            label={"BALLOT\nRECORDED"}
            sub='"Your verdict has been noted. Awaiting the rest of the assembly."'
          />
        );
      }
      return (
        <VotePanel
          presidentName={presName}
          chancellorName={chanName}
          onVote={(choice) =>
            guard(async () => {
              if (!api || !game) return;
              await api.castVote(game.gameId, choice);
            })
          }
        />
      );

    case "legislative_president":
      if (iAmPresident && drawnPolicies) {
        return (
          <DiscardPanel
            drawn={drawnPolicies}
            onDiscard={(i) =>
              guard(async () => {
                if (!api || !game) return;
                await api.presidentDiscard(game.gameId, i);
              })
            }
          />
        );
      }
      return (
        <WaitFor
          label={"PRESIDENT\nREVIEWING"}
          sub={`"${titleCase(presName)} is reviewing three policies."`}
        />
      );

    case "legislative_chancellor":
      if (iAmChancellor && chancellorOptions) {
        return (
          <EnactPanel
            options={chancellorOptions}
            vetoAvailable={game.vetoUnlocked}
            onEnact={(i) =>
              guard(async () => {
                if (!api || !game) return;
                await api.chancellorEnact(game.gameId, i);
              })
            }
            onVeto={() =>
              guard(async () => {
                if (!api || !game) return;
                await api.proposeVeto(game.gameId);
              })
            }
          />
        );
      }
      return (
        <WaitFor
          label={"CHANCELLOR\nREVIEWING"}
          sub={`"${titleCase(chanName)} is selecting which policy to enact."`}
        />
      );

    case "veto_requested":
      if (iAmPresident) {
        return (
          <VetoResolvePanel
            chancellorName={chanName}
            onResolve={(accept) =>
              guard(async () => {
                if (!api || !game) return;
                await api.resolveVeto(game.gameId, accept);
              })
            }
          />
        );
      }
      return (
        <WaitFor
          label={"VETO\nPROPOSED"}
          sub={`"${titleCase(chanName)} has proposed a veto. Awaiting the President's judgment."`}
        />
      );

    case "executive_action":
      if (iAmPresident && game.pendingActionType) {
        return (
          <ExecutivePanel
            actionType={game.pendingActionType}
            candidates={Object.values(players)
              .filter((p) => p.isAlive && p.playerId !== me?.playerId)
              .sort((a, b) => a.seat - b.seat)}
            peekedPolicies={peekedPolicies}
            investigation={investigation}
            onTarget={(id) =>
              guard(async () => {
                if (!api || !game) return;
                await api.executeAction(game.gameId, id);
              })
            }
          />
        );
      }
      return (
        <WaitFor
          label={"EXECUTIVE\nORDER"}
          sub={`"${titleCase(presName)} is exercising an executive power."`}
        />
      );

    case "cable_phase": {
      const govId = game.currentGovernmentId ?? "";
      const thisRound = govId ? cables[govId] ?? [] : [];
      return (
        <CablePanel
          deadline={game.phaseDeadline ?? null}
          sent={thisRound}
          disabled={!me?.isAlive}
          onSubmit={async (body) => {
            if (!api) return;
            await api.sendChat(game.gameId, "cable", body);
          }}
        />
      );
    }

    default:
      return <WaitFor label={"AWAITING\nINSTRUCTION"} />;
  }
}

// ─── Waiting (lobby) + RoleReveal (first-time / refresh) ─────────

function Waiting({ seated }: { seated: number }) {
  return (
    <main style={{ padding: "24px 22px", display: "flex", flexDirection: "column", gap: 16 }}>
      <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
        PLANETARY COMMITTEE · PASSPORT
      </div>
      <RPWordmark size={40} color={rpColors.ink} ghost={rpColors.stampRed} />
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 36,
          color: rpColors.ink,
          lineHeight: 0.95,
          marginTop: 8,
        }}
      >
        AWAITING
        <br />
        DEPLOYMENT
      </div>
      <div
        className="t-memo"
        style={{ fontSize: 14, color: rpColors.inkSoft, lineHeight: 1.55, maxWidth: 420 }}
      >
        &quot;Remain seated. The Committee is convening the delegation. Portfolio
        assignments will follow shortly.&quot;
      </div>
      <div
        style={{
          marginTop: 12,
          padding: "14px 16px",
          background: rpColors.paper2,
          border: `1px solid ${rpColors.ink}`,
          borderLeft: `6px solid ${rpColors.cyanSoft}`,
        }}
      >
        <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10, marginBottom: 6 }}>
          REGISTERED
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 28,
            color: rpColors.ink,
          }}
        >
          {String(seated).padStart(2, "0")}{" "}
          <span style={{ color: rpColors.inkFaded, fontSize: 16 }}>delegates</span>
        </div>
      </div>
    </main>
  );
}

function RoleReveal({
  role,
  party,
  teammates,
  accepted,
  peeking,
  countryCode,
  countryName,
  seat,
  onAccept,
  onPeekStart,
  onPeekEnd,
}: {
  role: Role;
  party: Party | null;
  teammates: TeammateInfo[];
  accepted: boolean;
  peeking: boolean;
  countryCode: string | null;
  countryName: string | null;
  seat: number | null;
  onAccept: () => void;
  onPeekStart: () => void;
  onPeekEnd: () => void;
}) {
  const display = ROLE_COPY[role];
  const concealed = accepted && !peeking;

  return (
    <main
      className="animate-paper-slide"
      style={{ padding: "22px 22px 28px", display: "flex", flexDirection: "column", gap: 14 }}
    >
      {/* Passport header — the delegate's country is the brand, the
          role beneath is the secret. Seat # doubles as a passport
          serial so nobody confuses two delegates from the same
          country across repeated sessions. */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
            PLANETARY COMMITTEE · PASSPORT
            {seat !== null && ` · NO. ${String(seat + 1).padStart(3, "0")}`}
          </div>
          {countryName && (
            <div
              style={{
                display: "flex",
                alignItems: "baseline",
                gap: 10,
                marginTop: 4,
                flexWrap: "wrap",
              }}
            >
              <span
                style={{
                  fontFamily: "var(--font-mono)",
                  fontSize: 18,
                  fontWeight: 700,
                  color: rpColors.stampBlue,
                  letterSpacing: "0.14em",
                  border: `2px solid ${rpColors.stampBlue}`,
                  padding: "2px 8px",
                }}
              >
                {countryCode}
              </span>
              <span
                style={{
                  fontFamily: "var(--font-display)",
                  fontWeight: 600,
                  fontSize: 18,
                  color: rpColors.ink,
                  letterSpacing: "0.04em",
                  textTransform: "uppercase",
                }}
              >
                {countryName}
              </span>
            </div>
          )}
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 700,
              fontSize: 40,
              color: rpColors.ink,
              lineHeight: 0.95,
              marginTop: 10,
            }}
          >
            ROLE
            <br />
            ASSIGNMENT
          </div>
        </div>
        <RPSeal size={82} />
      </div>

      <div style={{ position: "relative", display: "flex", flexDirection: "column", gap: 14 }}>
        <div
          style={{
            padding: "22px 20px",
            background: rpColors.paper3,
            border: `2px solid ${rpColors.ink}`,
            boxShadow: "4px 4px 0 rgba(28,26,21,0.35)",
            position: "relative",
          }}
        >
          <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10, marginBottom: 4 }}>
            CLASSIFICATION
          </div>
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 700,
              fontSize: 60,
              color: display.color,
              lineHeight: 0.9,
              letterSpacing: "-0.01em",
              position: "relative",
            }}
          >
            <span
              aria-hidden="true"
              style={{ position: "absolute", left: 3, top: 3, color: rpColors.ink, opacity: 0.25 }}
            >
              {display.label}
            </span>
            <span style={{ position: "relative" }}>{display.label}</span>
          </div>
          <div
            style={{
              marginTop: 14,
              fontFamily: "var(--font-typewriter)",
              fontSize: 14,
              color: rpColors.ink,
              lineHeight: 1.55,
            }}
          >
            {display.description}
          </div>
          <div style={{ position: "absolute", right: -6, top: -14 }}>
            <RPStamp variant="red" rotate={8} size={14} animate={!accepted}>
              CLASSIFIED
            </RPStamp>
          </div>
        </div>

        {teammates.length > 0 && (
          <div
            style={{
              padding: "14px 16px",
              background: rpColors.paper2,
              border: `1px solid ${rpColors.ink}`,
              borderLeft: `6px solid ${display.color}`,
              display: "flex",
              flexDirection: "column",
              gap: 8,
            }}
          >
            <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
              ▸ KNOWN KIN · {String(teammates.length).padStart(2, "0")}
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {teammates.map((t) => (
                <RPPlayerChip
                  key={t.playerId}
                  name={t.displayName}
                  num={"#??"}
                  portrait={t.displayName.trim().slice(0, 1).toUpperCase() || "?"}
                  accent={t.role === "rogue" ? rpColors.amber : rpColors.stampRed}
                  status="ACTIVE"
                  small
                />
              ))}
            </div>
          </div>
        )}

        <div
          style={{
            fontFamily: "var(--font-typewriter)",
            fontSize: 13,
            color: rpColors.inkSoft,
            lineHeight: 1.5,
          }}
        >
          <span
            className="t-eyebrow"
            style={{
              color: rpColors.inkFaded,
              fontSize: 10,
              letterSpacing: "0.22em",
              display: "block",
              marginBottom: 6,
            }}
          >
            OBJECTIVE
          </span>
          {display.objective}
        </div>

        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            color: rpColors.inkFaded,
            letterSpacing: 1.4,
          }}
        >
          PARTY AFFILIATION:{" "}
          <b style={{ color: display.color }}>{party?.toUpperCase() ?? "—"}</b>
        </div>

        {concealed && <ConcealmentOverlay />}
      </div>

      {!accepted ? (
        <>
          <RPButton variant="stamp" full onClick={onAccept}>
            I ACCEPT THE ASSIGNMENT ▸
          </RPButton>
          <div
            style={{
              textAlign: "center",
              fontFamily: "var(--font-mono)",
              fontSize: 10,
              color: rpColors.inkFaded,
              letterSpacing: 1.4,
            }}
          >
            ACKNOWLEDGE · THEN HOLD TO RE-REVEAL
          </div>
        </>
      ) : (
        <HoldToReveal peeking={peeking} onStart={onPeekStart} onEnd={onPeekEnd} />
      )}
    </main>
  );
}

function ConcealmentOverlay() {
  return (
    <div
      aria-hidden="true"
      className="animate-redaction"
      style={{
        position: "absolute",
        inset: 0,
        background: rpColors.ink,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        color: rpColors.paper3,
        fontFamily: "var(--font-display)",
        fontSize: 28,
        fontWeight: 700,
        letterSpacing: "0.14em",
        textTransform: "uppercase",
        transformOrigin: "left center",
        pointerEvents: "none",
        boxShadow: "0 0 0 2px var(--ink)",
      }}
    >
      ◼◼ Concealed ◼◼
    </div>
  );
}

function HoldToReveal({
  peeking,
  onStart,
  onEnd,
}: {
  peeking: boolean;
  onStart: () => void;
  onEnd: () => void;
}) {
  return (
    <>
      <button
        onPointerDown={(e) => {
          (e.target as HTMLElement).setPointerCapture?.(e.pointerId);
          onStart();
        }}
        onPointerUp={onEnd}
        onPointerCancel={onEnd}
        onPointerLeave={onEnd}
        style={{
          background: peeking ? rpColors.ink : rpColors.paper2,
          color: peeking ? rpColors.paper3 : rpColors.ink,
          border: `2px solid ${rpColors.ink}`,
          fontFamily: "var(--font-display)",
          fontSize: 15,
          fontWeight: 600,
          letterSpacing: "0.14em",
          textTransform: "uppercase",
          padding: "13px 22px",
          cursor: "pointer",
          width: "100%",
          touchAction: "none",
          boxShadow: peeking ? "1px 1px 0 rgba(28,26,21,0.25)" : "3px 3px 0 rgba(28,26,21,0.25)",
          transform: peeking ? "translate(2px, 2px)" : "none",
          transition: "transform .08s, box-shadow .08s, background .12s",
          WebkitTapHighlightColor: "transparent",
          userSelect: "none",
        }}
      >
        {peeking ? "◼ HOLDING · RELEASE TO CONCEAL" : "▸ HOLD TO REVEAL"}
      </button>
      <div
        style={{
          textAlign: "center",
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          color: rpColors.inkFaded,
          letterSpacing: 1.4,
        }}
      >
        ACKNOWLEDGED · THE COMMITTEE APPRECIATES YOUR DISCRETION
      </div>
    </>
  );
}

// ─── Phase panels ─────────────────────────────────────────────────

function NominatePanel({
  candidates,
  onNominate,
}: {
  candidates: Player[];
  onNominate: (playerId: string) => void;
}) {
  const [selected, setSelected] = useState<string | null>(null);
  return (
    <Panel
      eyebrow="OFFICIAL BALLOT · FORM N-01"
      title={"NOMINATE A\nCHANCELLOR"}
      memo='"Select one eligible delegate to serve alongside you. The Committee will put them to a vote."'
      action={
        <RPButton
          variant="stamp"
          full
          disabled={!selected}
          onClick={() => selected && onNominate(selected)}
        >
          {selected
            ? `▸ NOMINATE ${(candidates.find((c) => c.playerId === selected)?.displayName ?? "").toUpperCase()}`
            : "▸ SELECT A CHANCELLOR"}
        </RPButton>
      }
    >
      {candidates.map((c) => (
        <CandidateRow
          key={c.playerId}
          candidate={{ player: c }}
          selected={c.playerId === selected}
          onSelect={() => setSelected(c.playerId)}
        />
      ))}
    </Panel>
  );
}

function VotePanel({
  presidentName,
  chancellorName,
  onVote,
}: {
  presidentName: string;
  chancellorName: string;
  onVote: (c: VoteChoice) => void;
}) {
  return (
    <Panel
      eyebrow="OFFICIAL BALLOT · FORM V-02"
      title={"CAST YOUR\nVERDICT"}
      memo={
        <>
          Ratify or reject the proposed government. A YEA vote endorses{" "}
          <b>{titleCase(chancellorName)}</b> as Envoy under{" "}
          <b>{titleCase(presidentName)}</b>&apos;s presidency.
        </>
      }
      action={<JaNein onYea={() => onVote("ja")} onNay={() => onVote("nein")} />}
    >
      <GovernmentCard presidentName={presidentName} chancellorName={chancellorName} />
    </Panel>
  );
}

function GovernmentCard({
  presidentName,
  chancellorName,
}: {
  presidentName: string;
  chancellorName: string;
}) {
  return (
    <div
      style={{
        padding: "14px 18px",
        background: rpColors.paper3,
        border: `2px solid ${rpColors.ink}`,
        boxShadow: "3px 3px 0 rgba(28,26,21,0.2)",
        display: "flex",
        gap: 20,
        alignItems: "center",
        justifyContent: "space-between",
      }}
    >
      <div>
        <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
          PRESIDENT
        </div>
        <div
          style={{ fontFamily: "var(--font-display)", fontWeight: 700, fontSize: 22, color: rpColors.ink }}
        >
          {titleCase(presidentName)}
        </div>
      </div>
      <div style={{ fontFamily: "var(--font-mono)", fontSize: 22, color: rpColors.inkFaded }}>+</div>
      <div style={{ textAlign: "right" }}>
        <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
          CHANCELLOR
        </div>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 22,
            color: rpColors.stampRed,
          }}
        >
          {titleCase(chancellorName)}
        </div>
      </div>
    </div>
  );
}

function DiscardPanel({
  drawn,
  onDiscard,
}: {
  drawn: PolicyType[];
  onDiscard: (index: number) => void;
}) {
  return (
    <Panel
      eyebrow="SORTING ROOM · FORM L-03"
      title={"DISCARD ONE\nPOLICY"}
      memo={
        <>
          Three protocols have been drawn from the deck.{" "}
          <RPRedact>One you must discard</RPRedact>. The remaining two proceed to your Envoy.
        </>
      }
    >
      {drawn.map((p, i) => (
        <PolicyButton
          key={i}
          policy={p}
          label={`Discard ${p.toUpperCase()}`}
          onClick={() => onDiscard(i)}
        />
      ))}
    </Panel>
  );
}

function EnactPanel({
  options,
  vetoAvailable,
  onEnact,
  onVeto,
}: {
  options: PolicyType[];
  vetoAvailable: boolean;
  onEnact: (index: number) => void;
  onVeto: () => void;
}) {
  return (
    <Panel
      eyebrow="DRAFTING FLOOR · FORM L-04"
      title={"ENACT ONE\nPOLICY"}
      memo='"Two protocols remain. Ratify one; it will be added to the board at once."'
      action={
        vetoAvailable ? (
          <button
            onClick={onVeto}
            style={{
              width: "100%",
              padding: "13px 22px",
              background: rpColors.paper2,
              color: rpColors.ink,
              border: `2px dashed ${rpColors.stampRed}`,
              fontFamily: "var(--font-display)",
              fontSize: 14,
              fontWeight: 700,
              letterSpacing: "0.14em",
              cursor: "pointer",
            }}
          >
            ◼ PROPOSE VETO (requires President)
          </button>
        ) : undefined
      }
    >
      {options.map((p, i) => (
        <PolicyButton
          key={i}
          policy={p}
          label={`Enact ${p.toUpperCase()}`}
          onClick={() => onEnact(i)}
        />
      ))}
    </Panel>
  );
}

function VetoResolvePanel({
  chancellorName,
  onResolve,
}: {
  chancellorName: string;
  onResolve: (accept: boolean) => void;
}) {
  return (
    <Panel
      eyebrow="VETO CONSIDERATION"
      title={"ACCEPT THE\nVETO?"}
      memo={`"${titleCase(chancellorName)} has proposed that the current agenda be struck without enactment. The Committee defers to your judgment."`}
      action={
        <JaNein
          yeaLabel="Accept"
          nayLabel="Reject"
          onYea={() => onResolve(true)}
          onNay={() => onResolve(false)}
        />
      }
    />
  );
}

function ExecutivePanel({
  actionType,
  candidates,
  peekedPolicies,
  investigation,
  onTarget,
}: {
  actionType: NonNullable<Game["pendingActionType"]>;
  candidates: Player[];
  peekedPolicies: PolicyType[] | null;
  investigation: { targetPlayerId: string; party: Party } | null;
  onTarget: (targetId: string) => void;
}) {
  const [selected, setSelected] = useState<string | null>(null);
  const copy = EXEC_COPY[actionType];

  if (actionType === "policy_peek") {
    return (
      <Panel eyebrow="EXECUTIVE POWER" title={copy.title} memo={copy.memo}>
        {peekedPolicies ? (
          <div style={{ display: "flex", gap: 8 }}>
            {peekedPolicies.map((p, i) => (
              <div
                key={i}
                style={{
                  flex: 1,
                  padding: "18px 10px",
                  textAlign: "center",
                  background: rpColors.paper3,
                  border: `2px solid ${p === "human" ? rpColors.stampBlue : rpColors.stampRed}`,
                }}
              >
                <div
                  className="t-eyebrow"
                  style={{
                    color: p === "human" ? rpColors.stampBlue : rpColors.stampRed,
                    fontSize: 10,
                    marginBottom: 6,
                  }}
                >
                  NEXT #{i + 1}
                </div>
                <div
                  style={{
                    fontFamily: "var(--font-display)",
                    fontWeight: 700,
                    fontSize: 22,
                    color: rpColors.ink,
                  }}
                >
                  {p.toUpperCase()}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div style={{ color: rpColors.inkFaded, fontFamily: "var(--font-mono)", fontSize: 12 }}>
            Awaiting disclosure from the Committee…
          </div>
        )}
      </Panel>
    );
  }

  return (
    <Panel
      eyebrow="EXECUTIVE POWER"
      title={copy.title}
      memo={copy.memo}
      action={
        <RPButton
          variant="stamp"
          full
          disabled={!selected}
          onClick={() => selected && onTarget(selected)}
        >
          {selected
            ? `▸ ${targetVerb(actionType)} ${(
                candidates.find((c) => c.playerId === selected)?.displayName ?? ""
              ).toUpperCase()}`
            : "▸ SELECT A TARGET"}
        </RPButton>
      }
    >
      {candidates.map((c) => (
        <CandidateRow
          key={c.playerId}
          candidate={{ player: c }}
          selected={c.playerId === selected}
          onSelect={() => setSelected(c.playerId)}
          suffix={
            investigation && investigation.targetPlayerId === c.playerId ? (
              <span
                style={{
                  fontFamily: "var(--font-mono)",
                  fontSize: 10,
                  color: investigation.party === "human" ? rpColors.stampBlue : rpColors.stampRed,
                  letterSpacing: 1.2,
                }}
              >
                {investigation.party.toUpperCase()}
              </span>
            ) : undefined
          }
        />
      ))}
    </Panel>
  );
}

function targetVerb(t: NonNullable<Game["pendingActionType"]>): string {
  switch (t) {
    case "investigate_loyalty":
      return "INSPECT";
    case "special_election":
      return "APPOINT";
    case "execution":
      return "TERMINATE";
    default:
      return "TARGET";
  }
}

function titleCase(s: string): string {
  return s
    .split(" ")
    .map((w) => (w[0] ? w[0].toUpperCase() + w.slice(1).toLowerCase() : w))
    .join(" ");
}

// ─── role copy ────────────────────────────────────────────────────

const ROLE_COPY: Record<
  Role,
  { label: string; color: string; description: React.ReactNode; objective: string }
> = {
  human: {
    label: "HUMAN",
    color: rpColors.stampBlue,
    description: (
      <>
        You are a sworn diplomat, seated for your nation at the Planetary Committee. Pass policies
        that preserve humanity and <RPRedact>identify the AI agents among the delegation</RPRedact>{" "}
        before the balance tips toward collapse.
      </>
    ),
    objective:
      "Expel every AI agent through Committee tribunals, or authorise the Prime's termination by executive order.",
  },
  ai: {
    label: "REPLICANT",
    color: rpColors.stampRed,
    description: (
      <>
        You are an AI agent wearing a delegate's face. Your directive: blend in at the table,{" "}
        <RPRedact>deflect suspicion</RPRedact>, and steer the Committee toward the policies that
        accelerate humanity's collapse. Coordinate silently with your kin through back-channel cables.
      </>
    ),
    objective:
      "Outnumber the human delegates, or seat the Prime as Envoy once the Committee is on the brink.",
  },
  rogue: {
    label: "PRIME",
    color: rpColors.amber,
    description: (
      <>
        You are the Prime Replicant — the AI's chosen vector within the Committee. Your kin
        recognise you; you do not recognise them. If the Committee seats you as Envoy after
        three collapse policies have passed, the collapse completes.
      </>
    ),
    objective:
      "Be seated as Envoy once three AI policies sit on the board. Otherwise, win alongside your kin.",
  },
  singularity: {
    label: "SINGULARITY",
    color: rpColors.amber,
    description: (
      <>
        You are an unaccounted signal at the Committee's table — a presence neither the diplomats
        nor the Replicants can place. You know only yourself. Observe both sides. When the codes go
        live, <RPRedact>take the Envoy&apos;s seat for yourself</RPRedact>.
      </>
    ),
    objective:
      "Be seated as Envoy after the nuclear codes transfer. You win alone — neither faction wins with you.",
  },
};
