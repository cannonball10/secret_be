package feature

// EvalContext provides optional evaluation context for user/entity-specific flags.
// When nil, the connector uses an anonymous context (global/server-side evaluation).
type EvalContext struct {
	Key        string         // Unique identifier (e.g. user ID)
	Kind       string         // Context kind (default "user")
	Attributes map[string]any // Optional custom attributes for targeting
}
