package secrethitler

import "testing"

func TestPartyFor(t *testing.T) {
	cases := []struct {
		role Role
		want Party
	}{
		{RoleHuman, PartyHuman},
		{RoleAI, PartyAI},
		{RoleRogue, PartyAI},
		{RoleSingularity, PartySingularity},
	}
	for _, c := range cases {
		if got := PartyFor(c.role); got != c.want {
			t.Errorf("PartyFor(%q) = %q, want %q", c.role, got, c.want)
		}
	}
}

func TestRoleDistributionWithSingularity(t *testing.T) {
	// Below the 6-player floor the table is unavailable — the vanilla
	// 5-player distribution would leave humans outnumbered.
	if _, _, _, _, ok := RoleDistributionWithSingularity(5); ok {
		t.Error("RoleDistributionWithSingularity(5) should be unavailable")
	}
	// For 6+ it must subtract exactly one liberal and seat one
	// Singularity — preserving the fascists+hitlers columns.
	for n := 6; n <= 10; n++ {
		baseLib, baseFas, baseHit, _ := RoleDistribution(n)
		gotLib, gotFas, gotHit, gotSing, ok := RoleDistributionWithSingularity(n)
		if !ok {
			t.Errorf("RoleDistributionWithSingularity(%d) not ok", n)
			continue
		}
		if gotLib != baseLib-1 || gotFas != baseFas || gotHit != baseHit || gotSing != 1 {
			t.Errorf("RoleDistributionWithSingularity(%d) = (%d,%d,%d,%d); want (%d,%d,%d,1)",
				n, gotLib, gotFas, gotHit, gotSing, baseLib-1, baseFas, baseHit)
		}
		if gotLib+gotFas+gotHit+gotSing != n {
			t.Errorf("RoleDistributionWithSingularity(%d) seats %d != %d", n, gotLib+gotFas+gotHit+gotSing, n)
		}
	}
}

func TestDefaultRulesValid(t *testing.T) {
	if err := DefaultRules().Validate(); err != nil {
		t.Fatalf("DefaultRules() should pass Validate: %v", err)
	}
}

func TestRulesValidate(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*RulesConfig)
		want bool // true ⇒ expect error
	}{
		{"default ok", func(r *RulesConfig) {}, false},
		{"deck too small for human win", func(r *RulesConfig) { r.HumanProtocolsInDeck = 3 }, true},
		{"codes transfer >= ai win", func(r *RulesConfig) { r.CodesTransferAt = r.AIPoliciesToWin }, true},
		{"singularity requires 6+ players", func(r *RulesConfig) { r.EnableSingularity = true; r.MinPlayers = 5 }, true},
		{"singularity at 6 ok", func(r *RulesConfig) { r.EnableSingularity = true; r.MinPlayers = 6 }, false},
		{"min > max", func(r *RulesConfig) { r.MinPlayers = 10; r.MaxPlayers = 5 }, true},
	}
	for _, c := range cases {
		rc := DefaultRules()
		c.mut(&rc)
		err := rc.Validate()
		if (err != nil) != c.want {
			t.Errorf("%s: got err=%v, want err=%v", c.name, err, c.want)
		}
	}
}

func TestDeckComposition(t *testing.T) {
	if LiberalPoliciesInDeck+FascistPoliciesInDeck != 17 {
		t.Errorf("standard deck has 17 policies, got %d", LiberalPoliciesInDeck+FascistPoliciesInDeck)
	}
}
