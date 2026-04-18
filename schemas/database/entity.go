package database

// EntityType identifies whether a deployment/experiment targets a Prompt or Agent.
type EntityType string

const (
	EntityType_Prompt EntityType = "PROMPT"
	EntityType_Agent  EntityType = "AGENT"
)
