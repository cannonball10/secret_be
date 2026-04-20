// Mirror of schemas/replicant/powers.go. The presidential-power
// schedule depends on the starting player count and which AI policy
// (1-indexed) was just enacted. The host TV uses this to *preview*
// the next power — "if the cabal enacts another AI policy, the
// president will be granted an INVESTIGATE LOYALTY" — so players see
// the stakes of the next enactment without needing to memorise the
// rulebook.
//
// When reordering or extending this schedule, keep in sync with the
// Go version. Tests on the Go side pin the expected results.

import type { ExecutiveActionType } from "./enums";

const NINE_TEN: readonly (ExecutiveActionType | "")[] = [
  "investigate_loyalty",
  "investigate_loyalty",
  "special_election",
  "execution",
  "execution",
];
const SEVEN_EIGHT: readonly (ExecutiveActionType | "")[] = [
  "",
  "investigate_loyalty",
  "special_election",
  "execution",
  "execution",
];
const FIVE_SIX: readonly (ExecutiveActionType | "")[] = [
  "",
  "",
  "policy_peek",
  "execution",
  "execution",
];

/** Return the presidential power triggered when the Nth AI policy is
 *  enacted, or `null` if no power fires. `aiPolicyNumber` is 1-indexed
 *  (1..5). Unknown player counts return null. */
export function powerFor(
  playerCount: number,
  aiPolicyNumber: number,
): ExecutiveActionType | null {
  if (aiPolicyNumber < 1 || aiPolicyNumber > 5) return null;
  let table: readonly (ExecutiveActionType | "")[] | null = null;
  if (playerCount >= 9 && playerCount <= 10) table = NINE_TEN;
  else if (playerCount >= 7 && playerCount <= 8) table = SEVEN_EIGHT;
  else if (playerCount >= 5 && playerCount <= 6) table = FIVE_SIX;
  if (!table) return null;
  const entry = table[aiPolicyNumber - 1];
  return entry ? (entry as ExecutiveActionType) : null;
}

/** Human-readable label for a power, matching the aesthetic used on
 *  the board (short, stamped). */
export function powerLabel(p: ExecutiveActionType | null): string {
  switch (p) {
    case "investigate_loyalty":
      return "INVESTIGATE";
    case "special_election":
      return "SPECIAL ELECTION";
    case "policy_peek":
      return "POLICY PEEK";
    case "execution":
      return "NUCLEAR STRIKE";
    case "top_deck":
      return "TOP DECK";
    default:
      return "—";
  }
}
