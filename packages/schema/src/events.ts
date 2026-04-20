// Envelope wire format — mirrors handlers/game/events.go payloads and
// handlers/game/emitter.go envelope shape. Every SSE frame from the
// server parses into one of these Envelope variants.

import { z } from "zod";
import {
  AudienceScope,
  ChatChannel,
  EventType,
  ExecutiveActionType,
  GamePhase,
  Party,
  PolicyType,
  ProgressReason,
  Role,
  VoteChoice,
  WinCondition,
} from "./enums";

// ─── audience ────────────────────────────────────────────────────

export const Audience = z.object({
  scope: AudienceScope,
  playerId: z.string().optional(),
});
export type Audience = z.infer<typeof Audience>;

// ─── individual payloads ─────────────────────────────────────────

export const GameCreatedPayload = z.object({
  joinCode: z.string(),
  hostUserId: z.string(),
});
export type GameCreatedPayload = z.infer<typeof GameCreatedPayload>;

export const PlayerJoinedPayload = z.object({
  playerId: z.string(),
  displayName: z.string(),
  seat: z.number(),
});
export type PlayerJoinedPayload = z.infer<typeof PlayerJoinedPayload>;

export const GameStartedPayload = z.object({
  playerCount: z.number(),
  initialPresidentSeat: z.number(),
  round: z.number(),
});
export type GameStartedPayload = z.infer<typeof GameStartedPayload>;

export const TeammateInfo = z.object({
  playerId: z.string(),
  displayName: z.string(),
  role: Role,
});
export type TeammateInfo = z.infer<typeof TeammateInfo>;

export const RoleAssignedPayload = z.object({
  role: Role,
  party: Party,
  teammates: z.array(TeammateInfo).optional(),
});
export type RoleAssignedPayload = z.infer<typeof RoleAssignedPayload>;

export const ChancellorNominatedPayload = z.object({
  presidentPlayerId: z.string(),
  chancellorPlayerId: z.string(),
  governmentId: z.string(),
  deadline: z.string().optional(),
  round: z.number(),
});
export type ChancellorNominatedPayload = z.infer<typeof ChancellorNominatedPayload>;

export const VoteCastPayload = z.object({
  governmentId: z.string(),
  playerId: z.string(),
});
export type VoteCastPayload = z.infer<typeof VoteCastPayload>;

export const ElectionResultPayload = z.object({
  governmentId: z.string(),
  passed: z.boolean(),
  jaVotes: z.number(),
  neinVotes: z.number(),
  votes: z.record(VoteChoice),
  reason: ProgressReason,
});
export type ElectionResultPayload = z.infer<typeof ElectionResultPayload>;

export const PoliciesDrawnPayload = z.object({
  governmentId: z.string(),
  policies: z.array(PolicyType),
});
export type PoliciesDrawnPayload = z.infer<typeof PoliciesDrawnPayload>;

export const PresidentDiscardedPayload = z.object({
  governmentId: z.string(),
  options: z.array(PolicyType).optional(),
});
export type PresidentDiscardedPayload = z.infer<typeof PresidentDiscardedPayload>;

export const ChancellorEnactedPayload = z.object({
  governmentId: z.string(),
  policy: PolicyType,
  humanPoliciesEnacted: z.number(),
  aiPoliciesEnacted: z.number(),
});
export type ChancellorEnactedPayload = z.infer<typeof ChancellorEnactedPayload>;

export const VetoProposedPayload = z.object({ governmentId: z.string() });
export type VetoProposedPayload = z.infer<typeof VetoProposedPayload>;

export const VetoResolvedPayload = z.object({
  governmentId: z.string(),
  accepted: z.boolean(),
});
export type VetoResolvedPayload = z.infer<typeof VetoResolvedPayload>;

export const ExecutiveActionPayload = z.object({
  actionId: z.string(),
  type: ExecutiveActionType,
  presidentPlayerId: z.string(),
  targetPlayerId: z.string().optional(),
});
export type ExecutiveActionPayload = z.infer<typeof ExecutiveActionPayload>;

export const InvestigateResultPayload = z.object({
  actionId: z.string(),
  targetPlayerId: z.string(),
  party: Party,
});
export type InvestigateResultPayload = z.infer<typeof InvestigateResultPayload>;

export const PolicyPeekPayload = z.object({
  actionId: z.string(),
  policies: z.array(PolicyType),
});
export type PolicyPeekPayload = z.infer<typeof PolicyPeekPayload>;

export const ElectionTrackerPayload = z.object({ tracker: z.number() });
export type ElectionTrackerPayload = z.infer<typeof ElectionTrackerPayload>;

export const TopDeckPayload = z.object({
  policy: PolicyType,
  humanPoliciesEnacted: z.number(),
  aiPoliciesEnacted: z.number(),
});
export type TopDeckPayload = z.infer<typeof TopDeckPayload>;

export const DeckReshuffledPayload = z.object({ remainingInDraw: z.number() });
export type DeckReshuffledPayload = z.infer<typeof DeckReshuffledPayload>;

export const PlayerExecutedPayload = z.object({
  playerId: z.string(),
  wasRogue: z.boolean(),
  executedBySeat: z.number(),
});
export type PlayerExecutedPayload = z.infer<typeof PlayerExecutedPayload>;

export const GameEndedPayload = z.object({
  winner: Party,
  winCondition: WinCondition,
});
export type GameEndedPayload = z.infer<typeof GameEndedPayload>;

export const ChatMessagePayload = z.object({
  messageId: z.string(),
  channel: ChatChannel,
  authorPlayerId: z.string(),
  authorDisplayName: z.string(),
  body: z.string(),
  sentAt: z.string(),
  governmentId: z.string().optional(),
  ack: z.boolean().optional(),
});
export type ChatMessagePayload = z.infer<typeof ChatMessagePayload>;

// CablePhaseOpenedPayload — server's signal to unlock cable compose
// on every eligible device. Deadline is RFC3339 and drives the
// mobile countdown.
export const CablePhaseOpenedPayload = z.object({
  governmentId: z.string(),
  deadline: z.string(),
});
export type CablePhaseOpenedPayload = z.infer<typeof CablePhaseOpenedPayload>;

// CablePhaseClosedPayload — server's signal to lock cable compose.
// `leakedMessageId` is populated when the narrator leaks a cable,
// and matches the `messageId` in the follow-up cable_leaked envelope.
export const CablePhaseClosedPayload = z.object({
  governmentId: z.string(),
  reason: ProgressReason,
  leakedMessageId: z.string().optional(),
  totalSubmissions: z.number(),
});
export type CablePhaseClosedPayload = z.infer<typeof CablePhaseClosedPayload>;

// CableLeakedPayload — the narrator's phase-end broadcast. When
// `silenced` is true the body/author are empty and the paired
// narrator_speak envelope says the Committee reviewed the traffic
// and found nothing worth flagging.
export const CableLeakedPayload = z.object({
  governmentId: z.string(),
  silenced: z.boolean(),
  messageId: z.string().optional(),
  authorPlayerId: z.string().optional(),
  author: z.string().optional(),
  body: z.string().optional(),
  subversionScore: z.number().optional(),
});
export type CableLeakedPayload = z.infer<typeof CableLeakedPayload>;

/**
 * NarratorSpeakPayload — the Committee's synthesised utterance.
 * `audioUrl` is a `data:audio/mpeg;base64,…` URL; host clients play
 * it directly (and may GET /api/v1/narrator/audio/:audioId as a
 * fallback if they need to re-fetch). Mobile clients may render only
 * `script` if they prefer silent transcripts.
 */
export const NarratorSpeakPayload = z.object({
  cueId: z.string(),
  cue: z.string(),
  script: z.string(),
  audioUrl: z.string(),
  audioId: z.string(),
  format: z.string().optional(),
  tookMs: z.number().optional(),
});
export type NarratorSpeakPayload = z.infer<typeof NarratorSpeakPayload>;

/**
 * Phase-change events reuse the election_result envelope type on the
 * wire (see handlers/game/engine.go:setPhase). Carries the current
 * presidentSeat so clients can pick up rotation.
 */
export const PhaseChangedPayload = z.object({
  from: GamePhase,
  to: GamePhase,
  reason: ProgressReason,
  deadline: z.string().optional(),
  presidentSeat: z.number(),
});
export type PhaseChangedPayload = z.infer<typeof PhaseChangedPayload>;

// ─── envelope ────────────────────────────────────────────────────

/**
 * The GameEvent record carried inside every Envelope.event. Free-form
 * `data` is a lower-fidelity copy of the typed payload — we prefer the
 * payload field in reducers and fall back on `data` only when a newer
 * server adds a payload type we haven't mapped yet.
 */
export const GameEvent = z.object({
  eventId: z.string(),
  gameId: z.string(),
  type: EventType,
  actorPlayerId: z.string().optional(),
  targetPlayerId: z.string().optional(),
  data: z.record(z.unknown()).optional(),
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type GameEvent = z.infer<typeof GameEvent>;

export const Envelope = z.object({
  gameId: z.string(),
  event: GameEvent,
  audience: Audience,
  payload: z.unknown().optional(),
});
export type Envelope = z.infer<typeof Envelope>;
