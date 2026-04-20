// Mirror of schemas/secrethitler/countries.go. The backend assigns
// each seat a Country at StartGame — but the lobby wants to show the
// per-seat flag *before* assignment happens (so incoming players see
// "seat 3 will play as France"). Duplicating the table here keeps the
// UI responsive without a round-trip.
//
// When adding or reordering entries, keep this in sync with the Go
// file — the engine's seating index into CommitteeRoster is the
// source of truth once a game starts.

export interface Country {
  code: string;
  name: string;
}

export const COMMITTEE_ROSTER: Country[] = [
  { code: "US", name: "United States" },
  { code: "GB", name: "United Kingdom" },
  { code: "FR", name: "France" },
  { code: "DE", name: "Germany" },
  { code: "JP", name: "Japan" },
  { code: "IN", name: "India" },
  { code: "CN", name: "China" },
  { code: "BR", name: "Brazil" },
  { code: "NG", name: "Nigeria" },
  { code: "CA", name: "Canada" },
];

/** Returns the pre-assignment country for a given seat. The backend
 *  stamps the same value onto the Player row at StartGame. */
export function countryForSeat(seat: number): Country | null {
  if (seat < 0 || seat >= COMMITTEE_ROSTER.length) return null;
  return COMMITTEE_ROSTER[seat] ?? null;
}
