// Enums — mirror schemas/replicant/replicant.go on the Go side.
//
// Every literal lives in exactly one place (here) so adding a new
// status value is one edit, not a manual sweep.

import { z } from "zod";

// Role vocabulary mirrors schemas/replicant/replicant.go. The
// "ai"/"rogue" tokens remain for wire compatibility; in-fiction the
// Replicant theme presents them as Replicant / Prime. "singularity"
// is the solo kingmaker faction, dealt only when the server's
// RulesConfig.EnableSingularity is on.
export const Role = z.enum(["human", "ai", "rogue", "singularity"]);
export type Role = z.infer<typeof Role>;

export const Party = z.enum(["human", "ai", "singularity"]);
export type Party = z.infer<typeof Party>;

export const PolicyType = z.enum(["human", "ai"]);
export type PolicyType = z.infer<typeof PolicyType>;

export const GameStatus = z.enum(["lobby", "in_progress", "completed", "abandoned"]);
export type GameStatus = z.infer<typeof GameStatus>;

export const GamePhase = z.enum([
  "lobby",
  "nomination",
  "cable_phase",
  "election",
  "legislative_president",
  "legislative_chancellor",
  "veto_requested",
  "executive_action",
  "game_over",
]);
export type GamePhase = z.infer<typeof GamePhase>;

export const VoteChoice = z.enum(["ja", "nein"]);
export type VoteChoice = z.infer<typeof VoteChoice>;

export const GovernmentStatus = z.enum([
  "proposed",
  "passed",
  "rejected",
  "enacted",
  "vetoed",
]);
export type GovernmentStatus = z.infer<typeof GovernmentStatus>;

export const ExecutiveActionType = z.enum([
  "investigate_loyalty",
  "special_election",
  "policy_peek",
  "execution",
  "top_deck",
]);
export type ExecutiveActionType = z.infer<typeof ExecutiveActionType>;

export const WinCondition = z.enum([
  "human_policies",
  "ai_policies",
  "rogue_elected_chancellor",
  "rogue_executed",
  "singularity_kingmaker",
]);
export type WinCondition = z.infer<typeof WinCondition>;

export const EventType = z.enum([
  "game_created",
  "player_joined",
  "player_left",
  "game_started",
  "roles_assigned",
  "chancellor_nominated",
  "vote_cast",
  "election_result",
  "policies_drawn",
  "president_discarded",
  "chancellor_enacted",
  "veto_proposed",
  "veto_resolved",
  "executive_action",
  "election_tracker_advanced",
  "top_deck_enacted",
  "deck_reshuffled",
  "player_executed",
  "game_ended",
  "chat_message",
  "narrator_speak",
  "cable_phase_opened",
  "cable_phase_closed",
  "cable_leaked",
]);
export type EventType = z.infer<typeof EventType>;

/**
 * Chat room scopes.
 * - "ai"     → replicant/rogue cabal whisper channel (deprecated in UI;
 *              backend still accepts it).
 * - "dm"     → direct message between two alive delegates. Server
 *              persists and whispers to both parties; recipientPlayerId
 *              on the payload identifies the other party.
 * - "cable"  → Cable Phase submissions. Server persists and whispers
 *              an Ack=true back to the sender; other players never
 *              see the content at send time. The phase-end LLM ranker
 *              picks one cable to broadcast via the narrator.
 */
export const ChatChannel = z.enum(["ai", "dm", "cable"]);
export type ChatChannel = z.infer<typeof ChatChannel>;

export const ProgressReason = z.enum(["action", "timeout", "forced", "all_voted"]);
export type ProgressReason = z.infer<typeof ProgressReason>;

export const AudienceScope = z.enum(["broadcast", "player"]);
export type AudienceScope = z.infer<typeof AudienceScope>;

export const DeviceRole = z.enum(["board", "player"]);
export type DeviceRole = z.infer<typeof DeviceRole>;
