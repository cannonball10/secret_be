package replicant

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

// RoleDistributionWithSingularity returns seating for the 3-faction
// variant: one Singularity seat drawn from the liberal allotment of
// the vanilla distribution. Only defined for 6+ player counts — the
// 5-player distribution leaves liberals too outnumbered once one seat
// is spent on the Singularity. Call RoleDistribution for vanilla.
func RoleDistributionWithSingularity(playerCount int) (liberals, fascists, hitlers, singularities int, ok bool) {
	if playerCount < 6 {
		return 0, 0, 0, 0, false
	}
	l, f, h, ok := RoleDistribution(playerCount)
	if !ok {
		return 0, 0, 0, 0, false
	}
	return l - 1, f, h, 1, true
}

// VetoUnlockThreshold is the fascist-policy count at which the chancellor
// and president may jointly veto an agenda.
const VetoUnlockThreshold = 5
