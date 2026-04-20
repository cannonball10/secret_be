// Types mirror the backend OpenAPI spec (docs/openapi.yaml). Keep this
// file in sync with schemas/secrethitler and handlers/game/events.go.

// ─── enums ──────────────────────────────────────────────────────────

export type Role = "human" | "ai" | "rogue";
export type Party = "human" | "ai";
export type PolicyType = "human" | "ai";

export type GameStatus = "lobby" | "in_progress" | "completed" | "abandoned";

export type GamePhase =
  | "lobby"
  | "nomination"
  | "election"
  | "legislative_president"
  | "legislative_chancellor"
  | "veto_requested"
  | "executive_action"
  | "game_over";

export type VoteChoice = "ja" | "nein";

export type GovernmentStatus =
  | "proposed"
  | "passed"
  | "rejected"
  | "enacted"
  | "vetoed";

export type ExecutiveActionType =
  | "investigate_loyalty"
  | "special_election"
  | "policy_peek"
  | "execution"
  | "top_deck";

export type WinCondition =
  | "human_policies"
  | "ai_policies"
  | "rogue_elected_chancellor"
  | "rogue_executed";

export type EventType =
  | "game_created"
  | "player_joined"
  | "player_left"
  | "game_started"
  | "roles_assigned"
  | "chancellor_nominated"
  | "vote_cast"
  | "election_result"
  | "policies_drawn"
  | "president_discarded"
  | "chancellor_enacted"
  | "veto_proposed"
  | "veto_resolved"
  | "executive_action"
  | "election_tracker_advanced"
  | "top_deck_enacted"
  | "deck_reshuffled"
  | "player_executed"
  | "game_ended";

export type ProgressReason = "action" | "timeout" | "forced" | "all_voted";

// ─── core models ────────────────────────────────────────────────────

export interface Timestamps {
  createdAt: string;
  updatedAt: string;
}

export interface Game extends Timestamps {
  gameId: string;
  joinCode: string;
  hostUserId: string;
  status: GameStatus;
  phase: GamePhase;
  round: number;
  presidentSeat: number;
  chancellorSeat?: number | null;
  previousPresidentSeat?: number | null;
  previousChancellorSeat?: number | null;
  specialElectionReturnSeat?: number | null;
  currentGovernmentId?: string;
  humanPoliciesEnacted: number;
  aiPoliciesEnacted: number;
  electionTracker: number;
  vetoUnlocked: boolean;
  playerCount: number;
  winner?: Party;
  winCondition?: WinCondition;
  phaseDeadline?: string | null;
  pendingActionType?: ExecutiveActionType;
  pendingActionId?: string;
  startedAt?: string | null;
  endedAt?: string | null;
}

export interface Player extends Timestamps {
  playerId: string;
  gameId: string;
  userId: string;
  displayName: string;
  seat: number;
  role?: Role;
  party?: Party;
  isHost: boolean;
  isAlive: boolean;
  isConnected: boolean;
  investigatedBySeats?: number[];
}

export interface Government extends Timestamps {
  governmentId: string;
  gameId: string;
  round: number;
  presidentPlayerId: string;
  chancellorPlayerId?: string;
  presidentSeat: number;
  chancellorSeat?: number | null;
  status: GovernmentStatus;
  isSpecialElection: boolean;
  jaVotes: number;
  neinVotes: number;
  enactedPolicy?: PolicyType;
  vetoProposed: boolean;
  vetoAccepted: boolean;
}

export interface GameEvent extends Timestamps {
  eventId: string;
  gameId: string;
  type: EventType;
  actorPlayerId?: string;
  targetPlayerId?: string;
  data?: Record<string, unknown>;
}

// ─── envelope (SSE wire format) ─────────────────────────────────────

export type AudienceScope = "broadcast" | "player";

export interface Audience {
  scope: AudienceScope;
  playerId?: string;
}

// Every payload is keyed by event.type. Keeping them discriminated
// lets a switch in the reducer stay type-safe.
export type Envelope =
  | Env<"game_created", GameCreatedPayload>
  | Env<"player_joined", PlayerJoinedPayload>
  | Env<"game_started", GameStartedPayload>
  | Env<"roles_assigned", RoleAssignedPayload | undefined>
  | Env<"chancellor_nominated", ChancellorNominatedPayload>
  | Env<"vote_cast", VoteCastPayload>
  | Env<"election_result", ElectionResultPayload>
  | Env<"policies_drawn", PoliciesDrawnPayload>
  | Env<"president_discarded", PresidentDiscardedPayload>
  | Env<"chancellor_enacted", ChancellorEnactedPayload>
  | Env<"veto_proposed", VetoProposedPayload>
  | Env<"veto_resolved", VetoResolvedPayload>
  | Env<"executive_action", ExecActionEnvPayload>
  | Env<"election_tracker_advanced", ElectionTrackerPayload>
  | Env<"top_deck_enacted", TopDeckPayload>
  | Env<"deck_reshuffled", DeckReshuffledPayload>
  | Env<"player_executed", PlayerExecutedPayload>
  | Env<"game_ended", GameEndedPayload | PhaseChangedPayload>;

interface Env<T extends EventType, P> {
  gameId: string;
  event: GameEvent & { type: T };
  audience: Audience;
  payload?: P;
}

// ─── payloads ───────────────────────────────────────────────────────

export interface GameCreatedPayload {
  joinCode: string;
  hostUserId: string;
}

export interface PlayerJoinedPayload {
  playerId: string;
  displayName: string;
  seat: number;
}

export interface GameStartedPayload {
  playerCount: number;
  initialPresidentSeat: number;
  round: number;
}

export interface TeammateInfo {
  playerId: string;
  displayName: string;
  role: Role;
}

export interface RoleAssignedPayload {
  role: Role;
  party: Party;
  teammates?: TeammateInfo[];
}

export interface ChancellorNominatedPayload {
  presidentPlayerId: string;
  chancellorPlayerId: string;
  governmentId: string;
  deadline?: string;
  round: number;
}

export interface VoteCastPayload {
  governmentId: string;
  playerId: string;
}

export interface ElectionResultPayload {
  governmentId: string;
  passed: boolean;
  jaVotes: number;
  neinVotes: number;
  votes: Record<string, VoteChoice>;
  reason: ProgressReason;
}

export interface PoliciesDrawnPayload {
  governmentId: string;
  policies: PolicyType[];
}

export interface PresidentDiscardedPayload {
  governmentId: string;
  options?: PolicyType[];
}

export interface ChancellorEnactedPayload {
  governmentId: string;
  policy: PolicyType;
  humanPoliciesEnacted: number;
  aiPoliciesEnacted: number;
}

export interface VetoProposedPayload {
  governmentId: string;
}

export interface VetoResolvedPayload {
  governmentId: string;
  accepted: boolean;
}

// The backend sends three different payload shapes on the
// executive_action event, depending on whether it's the public
// announcement, an Investigate reveal whisper, or a Policy Peek.
export type ExecActionEnvPayload =
  | ExecutiveActionPayload
  | InvestigateResultPayload
  | PolicyPeekPayload;

export interface ExecutiveActionPayload {
  actionId: string;
  type: ExecutiveActionType;
  presidentPlayerId: string;
  targetPlayerId?: string;
}

export interface InvestigateResultPayload {
  actionId: string;
  targetPlayerId: string;
  party: Party;
}

export interface PolicyPeekPayload {
  actionId: string;
  policies: PolicyType[];
}

export interface ElectionTrackerPayload {
  tracker: number;
}

export interface TopDeckPayload {
  policy: PolicyType;
  humanPoliciesEnacted: number;
  aiPoliciesEnacted: number;
}

export interface DeckReshuffledPayload {
  remainingInDraw: number;
}

export interface PlayerExecutedPayload {
  playerId: string;
  wasRogue: boolean;
  executedBySeat: number;
}

export interface GameEndedPayload {
  winner: Party;
  winCondition: WinCondition;
}

export interface PhaseChangedPayload {
  from: GamePhase;
  to: GamePhase;
  reason: ProgressReason;
  deadline?: string;
}
