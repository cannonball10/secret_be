package secrethitler

import "testing"

func TestPartyFor(t *testing.T) {
	cases := []struct {
		role Role
		want Party
	}{
		{RoleLiberal, PartyLiberal},
		{RoleFascist, PartyFascist},
		{RoleHitler, PartyFascist},
	}
	for _, c := range cases {
		if got := PartyFor(c.role); got != c.want {
			t.Errorf("PartyFor(%q) = %q, want %q", c.role, got, c.want)
		}
	}
}

func TestDeckComposition(t *testing.T) {
	if LiberalPoliciesInDeck+FascistPoliciesInDeck != 17 {
		t.Errorf("standard deck has 17 policies, got %d", LiberalPoliciesInDeck+FascistPoliciesInDeck)
	}
}
