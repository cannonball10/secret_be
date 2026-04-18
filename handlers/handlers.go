package handlers

import (
	"github.com/cannonball10/foundation/handlers/user"
)

// Deps composes all sub-handler dependency interfaces.
// Satisfied by dependencies.Dependencies.
type Deps interface {
	user.Deps
}

type Handlers struct {
	UserHandler *user.UserHandler
}

func NewHandlers(deps Deps) *Handlers {
	return &Handlers{
		UserHandler: user.NewUserHandler(deps),
	}
}
