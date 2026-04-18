package secrethitler

// PowerFor returns the executive action triggered when the Nth fascist
// policy is enacted, or "" if no power is triggered. N is 1-indexed.
// The power table depends on total player count.
//
// Reference (official rules):
//
//	5-6  players: -, -, peek,        exec, exec
//	7-8  players: -, inv, spec_elec, exec, exec
//	9-10 players: inv, inv, spec_elec, exec, exec
func PowerFor(playerCount, fascistPolicyNumber int) ExecutiveActionType {
	if fascistPolicyNumber < 1 || fascistPolicyNumber > 5 {
		return ""
	}
	switch {
	case playerCount >= 9 && playerCount <= 10:
		return [5]ExecutiveActionType{
			ActionInvestigateLoyalty,
			ActionInvestigateLoyalty,
			ActionSpecialElection,
			ActionExecution,
			ActionExecution,
		}[fascistPolicyNumber-1]
	case playerCount >= 7 && playerCount <= 8:
		return [5]ExecutiveActionType{
			"",
			ActionInvestigateLoyalty,
			ActionSpecialElection,
			ActionExecution,
			ActionExecution,
		}[fascistPolicyNumber-1]
	case playerCount >= 5 && playerCount <= 6:
		return [5]ExecutiveActionType{
			"",
			"",
			ActionPolicyPeek,
			ActionExecution,
			ActionExecution,
		}[fascistPolicyNumber-1]
	}
	return ""
}

// RoleDistribution returns how many liberals, fascists (excluding Hitler)
// and hitlers to seat for a given player count.
func RoleDistribution(playerCount int) (liberals, fascists, hitlers int, ok bool) {
	switch playerCount {
	case 5:
		return 3, 1, 1, true
	case 6:
		return 4, 1, 1, true
	case 7:
		return 4, 2, 1, true
	case 8:
		return 5, 2, 1, true
	case 9:
		return 5, 3, 1, true
	case 10:
		return 6, 3, 1, true
	}
	return 0, 0, 0, false
}

// VetoUnlockThreshold is the fascist-policy count at which the chancellor
// and president may jointly veto an agenda.
const VetoUnlockThreshold = 5
