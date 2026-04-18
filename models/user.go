package models

import (
	"github.com/cannonball10/foundation/schemas/authentication"
	"github.com/cannonball10/foundation/utils"
)

// UserKeys provides key construction for the User model.
var UserKeys = NewKeyBuilder("USER#", "USER#")

// UserGSI1Keys provides key construction for the User GSI1 (provider + auth ID).
var UserGSI1Keys = NewKeyBuilder("PROVIDER#", "AUTHENTICATION_ID#")

type UserRole string

const (
	UserRole_User  UserRole = "user"
	UserRole_Admin UserRole = "admin"
)

// User is a DynamoDB-backed model.
type User struct {
	Timestamps

	UserID                 string                                `json:"userId"`
	AuthenticationProvider authentication.AuthenticationProvider `json:"authenticationProvider"`
	AuthenticationID       string                                `json:"authenticationId"`
	Email                  string                                `json:"email"`
	DisplayName            string                                `json:"displayName"`
	Role                   UserRole                              `json:"role"`
}

// IsAdmin returns true if the user has the admin role.
func (u *User) IsAdmin() bool {
	return u.Role == UserRole_Admin
}

// NewUser creates a new User. If id is empty, a ULID will be auto-generated.
func NewUser(id *string, authenticationProvider authentication.AuthenticationProvider, authenticationID, email, displayName string, role UserRole) *User {
	// Auto-generate ULID if id is not provided
	if id == nil || *id == "" {
		ulid := utils.GenerateULID()
		id = &ulid
	}

	return &User{
		Timestamps:             NewTimestamps(),
		UserID:                 *id,
		AuthenticationProvider: authenticationProvider,
		AuthenticationID:       authenticationID,
		Email:                  email,
		DisplayName:            displayName,
		Role:                   role,
	}
}

func (u *User) PK() string {
	return UserKeys.PK(u.UserID)
}

func (u *User) SK() string {
	return UserKeys.SK(u.UserID)
}

func (u *User) GSIs() map[int]GSIKeyPair {
	return map[int]GSIKeyPair{
		1: UserGSI1Keys.Pair(string(u.AuthenticationProvider), u.AuthenticationID),
	}
}

func init() {
	RegisterModel(UserKeys, func() Model { return &User{} })
}
