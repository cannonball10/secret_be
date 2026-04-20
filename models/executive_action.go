package models

import (
	"github.com/cannonball10/foundation/schemas/replicant"
	"github.com/cannonball10/foundation/utils"
)

// ExecutiveActionKeys provides key construction for the ExecutiveAction model.
// PK: GAME#{gameId}, SK: ACTION#{actionId}
// actionId is a ULID so begins_with("ACTION#") returns actions in order.
var ExecutiveActionKeys = NewKeyBuilder("GAME#", "ACTION#")

// ExecutiveAction records a single use of a presidential power. For
// Investigate Loyalty the result is the target's party; for Policy Peek
// it is the three upcoming policies in order. Execution and Special
// Election just record the target.
type ExecutiveAction struct {
	Timestamps

	ActionID          string                           `json:"actionId"`
	GameID            string                           `json:"gameId"`
	Round             int                              `json:"round"`
	Type              replicant.ExecutiveActionType `json:"type"`
	PresidentPlayerID string                           `json:"presidentPlayerId"`
	TargetPlayerID    string                           `json:"targetPlayerId,omitempty"`

	// RevealedParty holds the party shown by an Investigate Loyalty action.
	RevealedParty replicant.Party `json:"revealedParty,omitempty"`
	// PeekedPolicies holds the three policies shown by a Policy Peek.
	PeekedPolicies []replicant.PolicyType `json:"-"`

	// Completed is true once the president finishes the action (some
	// actions, like Policy Peek, complete immediately; others wait on
	// client acknowledgement before advancing the phase).
	Completed bool `json:"completed"`
}

// NewExecutiveAction creates an ExecutiveAction in the not-yet-completed state.
func NewExecutiveAction(id *string, gameID string, round int, t replicant.ExecutiveActionType, presidentPlayerID string) *ExecutiveAction {
	if id == nil || *id == "" {
		ulid := utils.GenerateULID()
		id = &ulid
	}
	return &ExecutiveAction{
		Timestamps:        NewTimestamps(),
		ActionID:          *id,
		GameID:            gameID,
		Round:             round,
		Type:              t,
		PresidentPlayerID: presidentPlayerID,
	}
}

func (a *ExecutiveAction) PK() string { return ExecutiveActionKeys.PK(a.GameID) }
func (a *ExecutiveAction) SK() string { return ExecutiveActionKeys.SK(a.ActionID) }

func (a *ExecutiveAction) GSIs() map[int]GSIKeyPair { return nil }

// Complete marks the action as finished.
func (a *ExecutiveAction) Complete() {
	a.Completed = true
	a.Touch()
}

func init() {
	RegisterModel(ExecutiveActionKeys, func() Model { return &ExecutiveAction{} })
}
