// Session, Player, Government, Vote — mirror models/*.go. Field names
// match the JSON tags on the Go side so we parse responses directly.

import { z } from "zod";
import {
  ExecutiveActionType,
  GamePhase,
  GameStatus,
  GovernmentStatus,
  Party,
  PolicyType,
  Role,
  VoteChoice,
  WinCondition,
} from "./enums";

const Timestamps = z.object({
  createdAt: z.string(),
  updatedAt: z.string(),
});

// CablePhaseMode mirrors schemas/replicant.CablePhaseMode.
export const CablePhaseMode = z.enum(["disabled", "every_round", "on_activation"]);
export type CablePhaseMode = z.infer<typeof CablePhaseMode>;

// RulesConfig — the tunable rule-set stamped onto a Game at create
// time. Mirrors schemas/replicant/rules.go; the host settings panel
// PUTs a full RulesConfig back via /host/rules to mutate a lobby's
// rules before StartGame.
export const RulesConfig = z.object({
  minPlayers: z.number(),
  maxPlayers: z.number(),
  humanProtocolsInDeck: z.number(),
  aiProtocolsInDeck: z.number(),
  humanPoliciesToWin: z.number(),
  aiPoliciesToWin: z.number(),
  codesTransferAt: z.number(),
  electionTrackerLimit: z.number(),
  vetoUnlockAt: z.number(),
  enableSingularity: z.boolean().optional(),
  cablePhaseMode: CablePhaseMode.optional(),
  cablePhaseDurationSec: z.number().optional(),
  cableLeakSilenceChance: z.number().optional(),
  anonymousCableLeaks: z.boolean().optional(),
  disableNarrator: z.boolean().optional(),
});
export type RulesConfig = z.infer<typeof RulesConfig>;

export const Game = Timestamps.extend({
  gameId: z.string(),
  joinCode: z.string(),
  hostUserId: z.string(),
  status: GameStatus,
  phase: GamePhase,
  round: z.number(),
  presidentSeat: z.number(),
  chancellorSeat: z.number().nullish(),
  previousPresidentSeat: z.number().nullish(),
  previousChancellorSeat: z.number().nullish(),
  specialElectionReturnSeat: z.number().nullish(),
  currentGovernmentId: z.string().optional(),
  humanPoliciesEnacted: z.number(),
  aiPoliciesEnacted: z.number(),
  electionTracker: z.number(),
  vetoUnlocked: z.boolean(),
  playerCount: z.number(),
  winner: Party.optional(),
  winCondition: WinCondition.optional(),
  phaseDeadline: z.string().nullish(),
  pendingActionType: ExecutiveActionType.optional(),
  pendingActionId: z.string().optional(),
  startedAt: z.string().nullish(),
  endedAt: z.string().nullish(),
  rules: RulesConfig.optional(),
});
export type Game = z.infer<typeof Game>;

/**
 * A session is the in-game Game record — we alias it here so app code
 * can talk about "sessions" (per the Replicant spec) while the Go
 * engine keeps using Game internally. Same shape.
 */
export const Session = Game;
export type Session = Game;

export const Player = Timestamps.extend({
  playerId: z.string(),
  gameId: z.string(),
  userId: z.string(),
  displayName: z.string(),
  seat: z.number(),
  role: Role.optional(),
  party: Party.optional(),
  /** ISO-3166-1 alpha-2 code of the delegation this player represents
   *  on the Planetary Committee. Assigned at StartGame. */
  countryCode: z.string().optional(),
  /** Full display name of the delegation (e.g. "United States"). */
  countryName: z.string().optional(),
  isHost: z.boolean(),
  isAlive: z.boolean(),
  isConnected: z.boolean(),
  investigatedBySeats: z.array(z.number()).optional(),
});
export type Player = z.infer<typeof Player>;

export const Government = Timestamps.extend({
  governmentId: z.string(),
  gameId: z.string(),
  round: z.number(),
  presidentPlayerId: z.string(),
  chancellorPlayerId: z.string().optional(),
  presidentSeat: z.number(),
  chancellorSeat: z.number().nullish(),
  status: GovernmentStatus,
  isSpecialElection: z.boolean(),
  jaVotes: z.number(),
  neinVotes: z.number(),
  enactedPolicy: PolicyType.optional(),
  vetoProposed: z.boolean(),
  vetoAccepted: z.boolean(),
});
export type Government = z.infer<typeof Government>;

export const Vote = z.object({
  gameId: z.string(),
  governmentId: z.string(),
  playerId: z.string(),
  choice: VoteChoice,
  castAt: z.string(),
});
export type Vote = z.infer<typeof Vote>;

/**
 * ChannelMsg — Phase 3 chat. Pre-defined here so other code can import
 * the type without waiting for the wire protocol to land.
 */
export const ChannelMsg = z.object({
  messageId: z.string(),
  gameId: z.string(),
  /** "all" for assembly-wide, "ai" for replicant cabal, etc. */
  channel: z.string(),
  authorPlayerId: z.string(),
  body: z.string(),
  sentAt: z.string(),
});
export type ChannelMsg = z.infer<typeof ChannelMsg>;
