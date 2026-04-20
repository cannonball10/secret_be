package secrethitler

// Country is one delegation on the Planetary Committee. Each seat
// at the table is assigned a distinct country at StartGame so every
// table plays as a plausible UN-style security-policy council.
type Country struct {
	// Code is ISO-3166-1 alpha-2 (two uppercase letters). Used as a
	// stable id for passport stamps + UI layout.
	Code string `json:"code"`
	// Name is the short, title-cased display name.
	Name string `json:"name"`
}

// CommitteeRoster is the 10-country pool the engine draws from at
// game start. Deliberately mixed: G7 core, BRICS representation, a
// populous-South entry, and one traditional non-aligned voice — the
// fiction is a security committee convened against a global AI
// threat, and seating needs to feel geopolitically plausible without
// centring any one bloc.
//
// Order matters: the first N entries of this slice seat the first N
// players. For a 5-player game you get US/UK/FR/DE/JP; for 10 you
// get all of them. Tune this list to change the table's flavour
// without touching the engine's seating logic.
var CommitteeRoster = []Country{
	{Code: "US", Name: "United States"},
	{Code: "GB", Name: "United Kingdom"},
	{Code: "FR", Name: "France"},
	{Code: "DE", Name: "Germany"},
	{Code: "JP", Name: "Japan"},
	{Code: "IN", Name: "India"},
	{Code: "CN", Name: "China"},
	{Code: "BR", Name: "Brazil"},
	{Code: "NG", Name: "Nigeria"},
	{Code: "CA", Name: "Canada"},
}

// CountryForSeat returns the Country assigned to a given seat. Returns
// a zero Country for seats outside the roster (shouldn't happen —
// game seating is bounded to [5, 10]).
func CountryForSeat(seat int) Country {
	if seat < 0 || seat >= len(CommitteeRoster) {
		return Country{}
	}
	return CommitteeRoster[seat]
}
