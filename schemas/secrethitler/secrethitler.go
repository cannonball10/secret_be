// Package secrethitler defines shared constants for the Secret Hitler game.
package secrethitler

// Role is a player's secret role assigned at game start.
type Role string

const (
	RoleLiberal Role = "liberal"
	RoleFascist Role = "fascist"
	RoleHitler  Role = "hitler"
)

// Party is a player's public party membership (Hitler's party is Fascist).
type Party string

const (
	PartyLiberal Party = "liberal"
	PartyFascist Party = "fascist"
)

// PartyFor returns the party alignment for a given role.
func PartyFor(r Role) Party {
	if r == RoleLiberal {
		return PartyLiberal
	}
	return PartyFascist
}

// PolicyType is the type of a policy card.
type PolicyType string

const (
	PolicyLiberal PolicyType = "liberal"
	PolicyFascist PolicyType = "fascist"
)

// Standard deck composition for Secret Hitler.
const (
	LiberalPoliciesInDeck = 6
	FascistPoliciesInDeck = 11

	// Enacted-policy thresholds for win conditions.
	LiberalWinPolicyCount = 5
	FascistWinPolicyCount = 6

	// Fascist policies enacted threshold after which electing Hitler
	// as Chancellor results in a fascist victory.
	HitlerChancellorThreshold = 3

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
	PhaseElection              GamePhase = "election"
	PhaseLegislativePresident  GamePhase = "legislative_president"
	PhaseLegislativeChancellor GamePhase = "legislative_chancellor"
	PhaseVetoRequested         GamePhase = "veto_requested"
	PhaseExecutiveAction       GamePhase = "executive_action"
	PhaseGameOver              GamePhase = "game_over"
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
	WinLiberalPolicies WinCondition = "liberal_policies"
	WinFascistPolicies WinCondition = "fascist_policies"
	WinHitlerElected   WinCondition = "hitler_elected_chancellor"
	WinHitlerExecuted  WinCondition = "hitler_executed"
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
)
