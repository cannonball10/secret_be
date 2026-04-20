// Package secrethitler defines shared constants for the Secret Hitler game.
package secrethitler

// Role is a player's secret role assigned at game start.
//
// Naming is retained from the Secret Hitler origin (Human / AI / Rogue)
// even as the Replicant theme renames them in-fiction to Human /
// Replicant / Prime. A fourth role, Singularity, is only dealt when
// RulesConfig.EnableSingularity is true — it's a solo kingmaker
// faction (see PartyFor and the WinSingularityKingmaker condition).
type Role string

const (
	RoleHuman       Role = "human"
	RoleAI          Role = "ai"          // in-fiction: Replicant
	RoleRogue       Role = "rogue"       // in-fiction: Prime
	RoleSingularity Role = "singularity" // solo faction; opt-in via rules
)

// Party is a player's faction alignment. The three-faction variant
// (Singularity) is only reachable when the ruleset enables it; vanilla
// games stay in the original 2-faction model.
type Party string

const (
	PartyHuman       Party = "human"
	PartyAI          Party = "ai" // in-fiction: Replicant cabal
	PartySingularity Party = "singularity"
)

// PartyFor returns the party alignment for a given role. Singularity
// is its own faction — it is not aligned with either the humans or
// the AI cabal, and wins alone under the kingmaker condition.
func PartyFor(r Role) Party {
	switch r {
	case RoleHuman:
		return PartyHuman
	case RoleSingularity:
		return PartySingularity
	default:
		return PartyAI
	}
}

// PolicyType is the type of a policy card.
type PolicyType string

const (
	PolicyHuman PolicyType = "human"
	PolicyAI PolicyType = "ai"
)

// Standard deck composition for Secret Hitler.
const (
	LiberalPoliciesInDeck = 6
	FascistPoliciesInDeck = 11

	// Enacted-policy thresholds for win conditions.
	HumanWinPolicyCount = 5
	AIWinPolicyCount = 6

	// Fascist policies enacted threshold after which electing Hitler
	// as Chancellor results in a fascist victory.
	RogueChancellorThreshold = 3

	// Election tracker limit: after 3 failed elections the top policy
	// is enacted automatically and term limits reset.
	ElectionTrackerLimit = 3

	// Minimum and maximum player counts.
	MinPlayers = 5
	MaxPlayers = 10
)

// GameStatus is the lifecycle state of a game.
type GameStatus string

const (
	GameStatusLobby      GameStatus = "lobby"
	GameStatusInProgress GameStatus = "in_progress"
	GameStatusCompleted  GameStatus = "completed"
	GameStatusAbandoned  GameStatus = "abandoned"
)

// GamePhase is the current turn phase during an in-progress game.
type GamePhase string

const (
	PhaseLobby                 GamePhase = "lobby"
	PhaseNomination            GamePhase = "nomination"
	PhaseCablePhase            GamePhase = "cable_phase"
	PhaseElection              GamePhase = "election"
	PhaseLegislativePresident  GamePhase = "legislative_president"
	PhaseLegislativeChancellor GamePhase = "legislative_chancellor"
	PhaseVetoRequested         GamePhase = "veto_requested"
	PhaseExecutiveAction       GamePhase = "executive_action"
	PhaseGameOver              GamePhase = "game_over"
)

// CablePhaseMode controls when the Cable Phase interstitial runs
// between nomination and election. Set on RulesConfig.
type CablePhaseMode string

const (
	// CableModeDisabled skips the phase entirely (vanilla Secret
	// Hitler flow: nomination → election directly).
	CableModeDisabled CablePhaseMode = "disabled"
	// CableModeEveryRound inserts a Cable Phase after every
	// chancellor nomination.
	CableModeEveryRound CablePhaseMode = "every_round"
	// CableModeOnActivation inserts a Cable Phase only when a power
	// activates it — e.g. a Wiretap. Reserved for future work.
	CableModeOnActivation CablePhaseMode = "on_activation"
)

// VoteChoice is a player's vote on a proposed government.
type VoteChoice string

const (
	VoteJa   VoteChoice = "ja"
	VoteNein VoteChoice = "nein"
)

// GovernmentStatus is the lifecycle of a proposed government.
type GovernmentStatus string

const (
	GovernmentStatusProposed GovernmentStatus = "proposed"
	GovernmentStatusPassed   GovernmentStatus = "passed"
	GovernmentStatusRejected GovernmentStatus = "rejected"
	GovernmentStatusEnacted  GovernmentStatus = "enacted"
	GovernmentStatusVetoed   GovernmentStatus = "vetoed"
)

// ExecutiveActionType is a presidential power triggered by fascist policies.
type ExecutiveActionType string

const (
	ActionInvestigateLoyalty ExecutiveActionType = "investigate_loyalty"
	ActionSpecialElection    ExecutiveActionType = "special_election"
	ActionPolicyPeek         ExecutiveActionType = "policy_peek"
	ActionExecution          ExecutiveActionType = "execution"
	// ActionTopDeck is a synthetic action used to record an automatic
	// policy enactment triggered by the election tracker.
	ActionTopDeck ExecutiveActionType = "top_deck"
)

// WinCondition is the reason a game ended.
type WinCondition string

const (
	WinHumanPolicies        WinCondition = "human_policies"
	WinAIPolicies           WinCondition = "ai_policies"
	WinRogueElected         WinCondition = "rogue_elected_chancellor"
	WinRogueExecuted        WinCondition = "rogue_executed"
	WinSingularityKingmaker WinCondition = "singularity_kingmaker"
)

// EventType identifies entries in the game event log.
type EventType string

const (
	EventGameCreated         EventType = "game_created"
	EventPlayerJoined        EventType = "player_joined"
	EventPlayerLeft          EventType = "player_left"
	EventGameStarted         EventType = "game_started"
	EventRolesAssigned       EventType = "roles_assigned"
	EventChancellorNominated EventType = "chancellor_nominated"
	EventVoteCast            EventType = "vote_cast"
	EventElectionResult      EventType = "election_result"
	EventPoliciesDrawn       EventType = "policies_drawn"
	EventPresidentDiscarded  EventType = "president_discarded"
	EventChancellorEnacted   EventType = "chancellor_enacted"
	EventVetoProposed        EventType = "veto_proposed"
	EventVetoResolved        EventType = "veto_resolved"
	EventExecutiveAction     EventType = "executive_action"
	EventElectionTracker     EventType = "election_tracker_advanced"
	EventTopDeckEnacted      EventType = "top_deck_enacted"
	EventDeckReshuffled      EventType = "deck_reshuffled"
	EventPlayerExecuted      EventType = "player_executed"
	EventGameEnded           EventType = "game_ended"
	EventChatMessage         EventType = "chat_message"
	EventNarratorSpeak       EventType = "narrator_speak"
	EventCablePhaseOpened    EventType = "cable_phase_opened"
	EventCablePhaseClosed    EventType = "cable_phase_closed"
	EventCableLeaked         EventType = "cable_leaked"
)
