// The state store is a pure reducer that the app feeds two kinds of
// inputs:
//
//   1. REST snapshots  — dispatch({type: "snapshot", …}) after a
//      successful GET /games/:id or POST create/join.
//   2. Stream envelopes — dispatch({type: "envelope", envelope}) for
//      every SSE frame the device receives.
//
// Keeping the reducer pure (no I/O) means the entire game view is a
// function of state, and we can test it offline by replaying an event
// log.

import type {
  Audience,
  ChancellorEnactedPayload,
  ChancellorNominatedPayload,
  ElectionResultPayload,
  ElectionTrackerPayload,
  Envelope,
  ExecutiveActionType,
  Game,
  GameEndedPayload,
  GameStartedPayload,
  Government,
  InvestigateResultPayload,
  Party,
  PlayerExecutedPayload,
  PlayerJoinedPayload,
  PoliciesDrawnPayload,
  PolicyPeekPayload,
  PolicyType,
  PresidentDiscardedPayload,
  RoleAssignedPayload,
  TopDeckPayload,
  VoteCastPayload,
  VoteChoice,
  VetoResolvedPayload,
  Player,
  GameEvent,
  TeammateInfo,
  Role,
} from "./types";

// ─── shape ──────────────────────────────────────────────────────────

export interface GameState {
  // Public, everyone sees this:
  game: Game | null;
  players: Record<string, Player>; // indexed by playerId
  currentGovernment: Government | null;
  votes: Record<string, VoteChoice>; // cleared each round
  votesCast: Record<string, true>; // just the set of who's voted (for spoiler-free tallies)
  tracker: number;
  liberal: number;
  fascist: number;
  drawPileRemaining: number | null; // null = unknown
  winner: Party | null;
  winCondition: string | null;
  log: GameEvent[]; // append-only event log

  // Private to the local device (populated by whisper envelopes):
  me: Me;
}

export interface Me {
  playerId: string | null;
  role: Role | null;
  party: Party | null;
  teammates: TeammateInfo[];
  // Legislative secrets:
  drawnPolicies: PolicyType[] | null; // whispered to president
  chancellorOptions: PolicyType[] | null; // whispered to chancellor
  // Power results:
  peekedPolicies: PolicyType[] | null;
  lastInvestigation: { targetPlayerId: string; party: Party } | null;
}

export const initialMe: Me = {
  playerId: null,
  role: null,
  party: null,
  teammates: [],
  drawnPolicies: null,
  chancellorOptions: null,
  peekedPolicies: null,
  lastInvestigation: null,
};

export const initialState: GameState = {
  game: null,
  players: {},
  currentGovernment: null,
  votes: {},
  votesCast: {},
  tracker: 0,
  liberal: 0,
  fascist: 0,
  drawPileRemaining: null,
  winner: null,
  winCondition: null,
  log: [],
  me: initialMe,
};

// ─── actions ────────────────────────────────────────────────────────

export type Action =
  // Bulk snapshot from REST (e.g. after join or GET /games/:id).
  | {
      type: "snapshot";
      game: Game;
      players?: Player[];
      mePlayerId?: string;
    }
  // Streamed envelope from SSE.
  | { type: "envelope"; envelope: Envelope }
  // User logs out / leaves.
  | { type: "reset" };

// ─── reducer ────────────────────────────────────────────────────────

export function reducer(state: GameState, action: Action): GameState {
  switch (action.type) {
    case "reset":
      return initialState;

    case "snapshot": {
      const players: Record<string, Player> = {};
      for (const p of action.players ?? []) {
        players[p.playerId] = p;
      }
      return {
        ...state,
        game: action.game,
        players: { ...state.players, ...players },
        tracker: action.game.electionTracker,
        liberal: action.game.liberalPoliciesEnacted,
        fascist: action.game.fascistPoliciesEnacted,
        me: action.mePlayerId
          ? { ...state.me, playerId: action.mePlayerId }
          : state.me,
      };
    }

    case "envelope":
      return applyEnvelope(state, action.envelope);
  }
}

// ─── envelope dispatch ──────────────────────────────────────────────

function applyEnvelope(state: GameState, env: Envelope): GameState {
  // Always keep the raw log for debugging and history screens.
  const next: GameState = { ...state, log: [...state.log, env.event] };

  // TS can't narrow a nested-discriminator union (env.event.type) across
  // env.payload, so each case casts once to the payload shape the
  // discriminator guarantees. The casts are safe because we match on
  // the event type immediately above.
  const p = env.payload as unknown;

  switch (env.event.type) {
    case "game_created":
      return next;

    case "player_joined":
      return handlePlayerJoined(next, p as PlayerJoinedPayload | undefined);

    case "game_started":
      return handleGameStarted(next, p as GameStartedPayload | undefined);

    case "roles_assigned":
      return handleRolesAssigned(next, env.audience, p as RoleAssignedPayload | undefined);

    case "chancellor_nominated":
      return handleChancellorNominated(next, p as ChancellorNominatedPayload | undefined);

    case "vote_cast":
      return handleVoteCast(next, p as VoteCastPayload | undefined);

    case "election_result":
      return handleElectionResult(next, p as ElectionResultPayload | undefined);

    case "policies_drawn":
      return handlePoliciesDrawn(next, env.audience, p as PoliciesDrawnPayload | undefined);

    case "president_discarded":
      return handlePresidentDiscarded(next, env.audience, p as PresidentDiscardedPayload | undefined);

    case "chancellor_enacted":
      return handleChancellorEnacted(next, p as ChancellorEnactedPayload | undefined);

    case "veto_proposed":
      return {
        ...next,
        currentGovernment: next.currentGovernment
          ? { ...next.currentGovernment, vetoProposed: true }
          : null,
      };

    case "veto_resolved":
      return handleVetoResolved(next, p as VetoResolvedPayload | undefined);

    case "executive_action":
      return handleExecutiveAction(next, env.audience, p as ExecActionPayloadAny | undefined);

    case "election_tracker_advanced":
      return handleTrackerAdvanced(next, p as ElectionTrackerPayload | undefined);

    case "top_deck_enacted":
      return handleTopDeck(next, p as TopDeckPayload | undefined);

    case "deck_reshuffled":
      return next;

    case "player_executed":
      return handlePlayerExecuted(next, p as PlayerExecutedPayload | undefined);

    case "game_ended":
      return handleGameEnded(next, p as GameEndedAnyPayload | undefined);
  }
}

// Narrow helper types for the two envelopes whose payload is itself a
// small union on the wire.
type ExecActionPayloadAny =
  | (Partial<InvestigateResultPayload> & { targetPlayerId?: string; party?: Party })
  | (Partial<PolicyPeekPayload> & { policies?: PolicyType[] })
  | { actionId: string; type?: ExecutiveActionType; presidentPlayerId?: string; targetPlayerId?: string };

type GameEndedAnyPayload =
  | GameEndedPayload
  | { from?: unknown; to?: unknown };

// ─── handlers (one per event) ───────────────────────────────────────

function handlePlayerJoined(
  s: GameState,
  p?: PlayerJoinedPayload,
): GameState {
  if (!p) return s;
  return {
    ...s,
    players: {
      ...s.players,
      [p.playerId]: {
        ...(s.players[p.playerId] ?? {}),
        playerId: p.playerId,
        displayName: p.displayName,
        seat: p.seat,
        isAlive: true,
        isConnected: true,
        isHost: false,
        gameId: s.game?.gameId ?? "",
        userId: s.players[p.playerId]?.userId ?? "",
        createdAt: s.players[p.playerId]?.createdAt ?? "",
        updatedAt: s.players[p.playerId]?.updatedAt ?? "",
      } as Player,
    },
  };
}

function handleGameStarted(s: GameState, p?: GameStartedPayload): GameState {
  if (!s.game || !p) return s;
  return {
    ...s,
    game: {
      ...s.game,
      status: "in_progress",
      phase: "nomination",
      playerCount: p.playerCount,
      presidentSeat: p.initialPresidentSeat,
    },
  };
}

function handleRolesAssigned(
  s: GameState,
  audience: Audience,
  p?: RoleAssignedPayload,
): GameState {
  // Only whispered role envelopes carry the private payload. Broadcast
  // "roles_assigned" (audience=broadcast) is just a signal to show the
  // reveal UI; it has no payload.
  if (audience.scope !== "player" || !p) return s;
  if (audience.playerId !== s.me.playerId) return s;
  return {
    ...s,
    me: {
      ...s.me,
      role: p.role,
      party: p.party,
      teammates: p.teammates ?? [],
    },
  };
}

function handleChancellorNominated(
  s: GameState,
  p?: ChancellorNominatedPayload,
): GameState {
  if (!s.game || !p) return s;
  const chancellor = Object.values(s.players).find(
    (pl) => pl.playerId === p.chancellorPlayerId,
  );
  return {
    ...s,
    game: {
      ...s.game,
      phase: "election",
      currentGovernmentId: p.governmentId,
      chancellorSeat: chancellor?.seat ?? null,
    },
    currentGovernment: {
      governmentId: p.governmentId,
      gameId: s.game.gameId,
      round: s.game.round,
      presidentPlayerId: p.presidentPlayerId,
      chancellorPlayerId: p.chancellorPlayerId,
      presidentSeat: s.game.presidentSeat,
      chancellorSeat: chancellor?.seat ?? null,
      status: "proposed",
      isSpecialElection: false,
      jaVotes: 0,
      neinVotes: 0,
      vetoProposed: false,
      vetoAccepted: false,
      createdAt: "",
      updatedAt: "",
    },
    votes: {},
    votesCast: {},
  };
}

function handleVoteCast(s: GameState, p?: VoteCastPayload): GameState {
  if (!p) return s;
  return { ...s, votesCast: { ...s.votesCast, [p.playerId]: true } };
}

function handleElectionResult(
  s: GameState,
  p?: ElectionResultPayload,
): GameState {
  if (!p) return s;
  return {
    ...s,
    votes: p.votes,
    currentGovernment: s.currentGovernment
      ? {
          ...s.currentGovernment,
          status: p.passed ? "passed" : "rejected",
          jaVotes: p.jaVotes,
          neinVotes: p.neinVotes,
        }
      : null,
  };
}

function handlePoliciesDrawn(
  s: GameState,
  audience: Audience,
  p?: PoliciesDrawnPayload,
): GameState {
  if (!s.game) return s;
  const next = { ...s, game: { ...s.game, phase: "legislative_president" as const } };
  if (audience.scope === "player" && audience.playerId === s.me.playerId && p) {
    next.me = { ...s.me, drawnPolicies: p.policies };
  }
  return next;
}

function handlePresidentDiscarded(
  s: GameState,
  audience: Audience,
  p?: PresidentDiscardedPayload,
): GameState {
  if (!s.game) return s;
  const next = {
    ...s,
    game: { ...s.game, phase: "legislative_chancellor" as const },
  };
  // Only the chancellor-whisper variant contains `options`.
  if (audience.scope === "player" && audience.playerId === s.me.playerId && p?.options) {
    next.me = {
      ...s.me,
      chancellorOptions: p.options,
      drawnPolicies: null,
    };
  }
  return next;
}

function handleChancellorEnacted(
  s: GameState,
  p?: ChancellorEnactedPayload,
): GameState {
  if (!p || !s.game) return s;
  return {
    ...s,
    liberal: p.liberalPoliciesEnacted,
    fascist: p.fascistPoliciesEnacted,
    tracker: 0,
    game: {
      ...s.game,
      liberalPoliciesEnacted: p.liberalPoliciesEnacted,
      fascistPoliciesEnacted: p.fascistPoliciesEnacted,
      electionTracker: 0,
    },
    currentGovernment: s.currentGovernment
      ? { ...s.currentGovernment, status: "enacted", enactedPolicy: p.policy }
      : null,
    // Clear our legislative secrets once the round wraps.
    me: { ...s.me, drawnPolicies: null, chancellorOptions: null },
  };
}

function handleVetoResolved(s: GameState, p?: VetoResolvedPayload): GameState {
  if (!s.currentGovernment || !p) return s;
  return {
    ...s,
    currentGovernment: {
      ...s.currentGovernment,
      vetoAccepted: p.accepted,
      status: p.accepted ? "vetoed" : s.currentGovernment.status,
    },
  };
}

function handleExecutiveAction(
  s: GameState,
  audience: Audience,
  p?: ExecActionPayloadAny,
): GameState {
  // Broadcast: announce the action. Whisper: receive a private
  // investigation or peek result.
  if (!p) return s;
  if (audience.scope === "player" && audience.playerId === s.me.playerId) {
    if ("party" in p && p.party && p.targetPlayerId) {
      return {
        ...s,
        me: {
          ...s.me,
          lastInvestigation: { targetPlayerId: p.targetPlayerId, party: p.party },
        },
      };
    }
    if ("policies" in p && Array.isArray(p.policies)) {
      return { ...s, me: { ...s.me, peekedPolicies: p.policies } };
    }
  }
  return s;
}

function handleTrackerAdvanced(
  s: GameState,
  p?: ElectionTrackerPayload,
): GameState {
  if (!p || !s.game) return s;
  return { ...s, tracker: p.tracker, game: { ...s.game, electionTracker: p.tracker } };
}

function handleTopDeck(s: GameState, p?: TopDeckPayload): GameState {
  if (!p || !s.game) return s;
  return {
    ...s,
    liberal: p.liberalPoliciesEnacted,
    fascist: p.fascistPoliciesEnacted,
    tracker: 0,
    game: {
      ...s.game,
      liberalPoliciesEnacted: p.liberalPoliciesEnacted,
      fascistPoliciesEnacted: p.fascistPoliciesEnacted,
      electionTracker: 0,
    },
  };
}

function handlePlayerExecuted(
  s: GameState,
  p?: PlayerExecutedPayload,
): GameState {
  if (!p) return s;
  const existing = s.players[p.playerId];
  if (!existing) return s;
  return {
    ...s,
    players: {
      ...s.players,
      [p.playerId]: { ...existing, isAlive: false },
    },
  };
}

function handleGameEnded(
  s: GameState,
  p?: GameEndedPayload | { from?: unknown; to?: unknown },
): GameState {
  if (!s.game) return s;
  // The envelope can carry either GameEndedPayload (winner + condition)
  // or a PhaseChangedPayload (if the event fired purely for the phase
  // transition). Only the former has the ending fields.
  if (p && "winner" in p) {
    return {
      ...s,
      game: { ...s.game, status: "completed", phase: "game_over", winner: p.winner, winCondition: p.winCondition },
      winner: p.winner,
      winCondition: p.winCondition,
    };
  }
  return {
    ...s,
    game: { ...s.game, phase: "game_over", status: "completed" },
  };
}

// ─── derived selectors ──────────────────────────────────────────────

// seatedPlayers returns players sorted by their seat — the order used
// by the board display for the round rotation.
export function seatedPlayers(s: GameState): Player[] {
  return Object.values(s.players).sort((a, b) => a.seat - b.seat);
}

export function currentPresident(s: GameState): Player | null {
  if (!s.game) return null;
  return seatedPlayers(s).find((p) => p.seat === s.game!.presidentSeat) ?? null;
}

export function currentChancellor(s: GameState): Player | null {
  if (!s.game || s.game.chancellorSeat == null) return null;
  return seatedPlayers(s).find((p) => p.seat === s.game!.chancellorSeat) ?? null;
}

export function isMePresident(s: GameState): boolean {
  const p = currentPresident(s);
  return !!p && p.playerId === s.me.playerId;
}

export function isMeChancellor(s: GameState): boolean {
  const c = currentChancellor(s);
  return !!c && c.playerId === s.me.playerId;
}
