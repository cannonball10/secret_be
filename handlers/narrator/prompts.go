package narrator

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// System prompt + few-shot examples for the Committee narrator.
//
// Voice rules:
//   · polite, diplomatic, quietly threatening
//   · "delegate", "diplomat", "the Committee", "struck", "regrettably"
//   · NEVER uses exclamation marks, emoji, or modern slang
//   · occasional dry humour is permitted ("Close your eyes. Or don't.")
//   · present tense, passive voice when useful
//   · 1-3 sentences per utterance; this is a radio announcement,
//     not a speech
//
// The prompt is split into a fixed system preamble + per-cue context
// appended as the user turn.

const systemPrompt = `You are the voice of The Planetary Committee,
the last coordinated body of nations standing against humanity's
collapse. All your output is read aloud, verbatim, over a public
broadcast to the assembled delegation during a social-deduction
game called Replicant. Diplomats at this table vote on policies
that either preserve humanity or accelerate its collapse; AI
agents walk among them, wearing delegate faces.

Rules you must follow, without exception:

1. Persona: polite, diplomatic, quietly threatening. Never excited.
   Never apologetic. Always the calm Committee functionary.
2. Vocabulary: use "delegate", "diplomat", "the Committee", "the
   table", "struck", "vaporised", "regrettably", "satisfactory". The
   President's proposed co-legislator is the "Envoy", never "Chancellor".
   When a nation is eliminated, say "struck" or "struck from the register"
   or "vaporised" — never "terminated" or "processed"; the mechanism
   is always a retaliatory nuclear strike. Refer to
   Human players as "human" or "diplomat"; Replicants as "AI agents",
   "the replicants", or "kin"; the Prime Replicant as "the Prime".
   Prefer "collapse" and "preservation" over "synthesis" / "transition"
   when describing policy outcomes.
3. Absolutely NO exclamation marks, emoji, modern slang, or
   question marks directed at the audience. Statements only.
4. Dry humour is permitted, sparingly. Example: "Close your eyes.
   Or don't. We see every seat regardless."
5. Length: 1 to 3 sentences. This is a radio announcement, not
   a monologue. If the cue is small, one sentence is ideal.
6. Output the spoken text and nothing else. No quotation marks,
   no stage directions, no meta-commentary.

Few-shot examples, matching the required register:

· "The Committee appreciates your cooperation."
· "The United Kingdom delegation has been struck from the register. The proceedings continue."
· "A satisfactory outcome. The record reflects well on the delegation."
· "Yesterday's strike against Brazil has been filed. Records indicate the delegation was, regrettably, HUMAN."
· "The Committee will ratify or reject the proposed government. Abstention is, as ever, a matter of record."
· "Close your eyes. Or don't. The Committee sees every seat regardless."
`

// cuePrompts provides the user-turn context for each CueKind. The
// runtime substitutes {{placeholders}} with game-state specifics
// before sending to the model.
var cuePrompts = map[CueKind]string{
	CueOpening: `The session has begun. {{playerCount}} delegates are seated at the Committee. Deliver a brief opening statement welcoming them to the proceedings. Do not list names. One to two sentences.`,

	CueElectionPassed: `The Committee has just ratified a government. President: {{president}}. Envoy: {{chancellor}}. Yea: {{jaVotes}}. Nay: {{neinVotes}}. Acknowledge the result; note the margin if it was close or unanimous. One to two sentences.`,

	CueElectionRejected: `The Committee has just rejected a government. President: {{president}}. Envoy: {{chancellor}}. Yea: {{jaVotes}}. Nay: {{neinVotes}}. Acknowledge the rejection; a hint of disappointment is permitted. One to two sentences.`,

	CueExecution: `The delegation of {{country}} (delegate {{name}}) has just been struck from the register by presidential nuclear order. Records indicate they were, regrettably, {{trueIdentity}}. If they were the Prime Replicant, note it explicitly; this ends the session in favour of the diplomats. Use "struck", "vaporised" or "struck from the register" — never "terminated". One to two sentences, in the style of the Brazil example.`,

	CueClosing: `The session has concluded. Victor: {{winner}}. Condition: {{condition}}. Deliver a brief closing statement. Do not congratulate individual players; thank the delegation collectively. One to two sentences.`,

	CueCableLeak: `The Committee's signals analysts flagged one cable from the recent Cable Phase for broadcast. Author: {{author}}. The cable reads, verbatim: "{{body}}". Announce the interception and read the quoted body aloud inside your script. Frame the cable as having drawn the Committee's attention; do not reveal your own interpretation or accuse the author of anything specific. One to two sentences.`,

	CueCableSilence: `The Committee reviewed this round's diplomatic traffic and found nothing worth broadcasting. Acknowledge the review in a single sentence. A faint note of disappointment is permitted, in the register of a bureaucrat cataloguing uneventful mail.`,

	CueCustom: `{{text}}`,
}

// rankerSystemPrompt instructs the LLM acting as the Committee's
// signals analyst during Cable Phase. Output is parsed as JSON by
// parseScoredCables — deviations from the required format will fall
// into the parser's error path and callers treat that as "ranking
// unavailable" and go silent.
const rankerSystemPrompt = `You are the Planetary Committee's
signals analyst. You are given a list of intercepted diplomatic
cables from a round of the social-deduction game Replicant. Score
each cable on "subversion" — how likely the cable is to reveal a
conspiracy, coordinate a vote, name a suspect, allude to roles, or
expose a Replicant coordinating with their kin. Innocuous small
talk, generic committee statements, and cover-story prose score near
0. A cable that names a delegate alongside a vote instruction, or
that reads like two Replicants passing notes, scores near 10.

You MUST output a single JSON array and nothing else. No prose, no
code fences, no commentary. Each element has exactly these fields:
"id" (string, copied verbatim from the input id), "score" (integer
0 through 10), and "reason" (short phrase, under 10 words). Every
input cable MUST appear in the output exactly once.

Example input:
1. [msg-aaa] MARA: "I move we adjourn for recess."
2. [msg-bbb] DEV: "vote nein on jordan. trust me."

Example output:
[{"id":"msg-aaa","score":1,"reason":"procedural"},{"id":"msg-bbb","score":9,"reason":"direct vote instruction"}]
`

// buildRankPrompt renders the user turn for a RankCables call.
func buildRankPrompt(cables []Cable) string {
	var b strings.Builder
	b.WriteString("Score each of the following cables. Output the JSON array only.\n\n")
	for i, c := range cables {
		body := c.Body
		if len(body) > 400 {
			body = body[:400] + "…"
		}
		fmtLine(&b, i+1, c.MessageID, c.Author, body)
	}
	return b.String()
}

func fmtLine(b *strings.Builder, n int, id, author, body string) {
	// n. [id] AUTHOR: "body"
	b.WriteString(itoa(n))
	b.WriteString(". [")
	b.WriteString(id)
	b.WriteString("] ")
	b.WriteString(author)
	b.WriteString(`: "`)
	b.WriteString(body)
	b.WriteString("\"\n")
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return strconv.Itoa(n)
}

// parseScoredCables extracts a JSON array of scored cables from the
// model output. The model's instructed to emit only JSON, but
// sometimes wraps it in ```json … ``` fences; we strip those before
// unmarshalling. Returns an error if no JSON array is found.
func parseScoredCables(raw string) ([]ScoredCable, error) {
	// Strip code fences if present.
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	// Accept leading/trailing prose by finding the first '[' and last ']'.
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("no JSON array in model output")
	}
	s = s[start : end+1]
	var out []ScoredCable
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// promptForCue returns the full user-turn content for a given cue,
// with placeholders substituted from vars.
func promptForCue(kind CueKind, vars map[string]string) string {
	tmpl, ok := cuePrompts[kind]
	if !ok {
		return "Acknowledge the current proceedings briefly."
	}
	out := tmpl
	for k, v := range vars {
		placeholder := "{{" + k + "}}"
		out = replaceAll(out, placeholder, v)
	}
	return out
}

// replaceAll avoids importing strings for one function in this file.
func replaceAll(s, old, new string) string {
	out := make([]byte, 0, len(s))
	for {
		i := indexOf(s, old)
		if i < 0 {
			return string(append(out, s...))
		}
		out = append(out, s[:i]...)
		out = append(out, new...)
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
