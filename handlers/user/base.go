package user

import (
	"github.com/cannonball10/foundation/connectors"
	"github.com/hibiken/asynq"
)

// Deps declares the capabilities the user handler needs.
// Satisfied by dependencies.Dependencies.
type Deps interface {
	GetConnectors() *connectors.Connectors
	GetTaskClient() *asynq.Client
}

type UserHandler struct {
	deps Deps
}

func NewUserHandler(deps Deps) *UserHandler {
	return &UserHandler{
		deps: deps,
	}
}
