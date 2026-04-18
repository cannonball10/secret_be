package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/authentication"
)

func TestNewUser(t *testing.T) {
	u := NewUser(nil, authentication.AuthenticationProvider_Clerk, "clerk_abc123", "alice@example.com", "Alice", UserRole_User)

	if u.UserID == "" {
		t.Fatal("expected UserID to be generated")
	}
	if u.AuthenticationProvider != authentication.AuthenticationProvider_Clerk {
		t.Errorf("expected AuthenticationProvider %q, got %q", authentication.AuthenticationProvider_Clerk, u.AuthenticationProvider)
	}
	if u.AuthenticationID != "clerk_abc123" {
		t.Errorf("expected AuthenticationID 'clerk_abc123', got %q", u.AuthenticationID)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected Email 'alice@example.com', got %q", u.Email)
	}
	if u.DisplayName != "Alice" {
		t.Errorf("expected DisplayName 'Alice', got %q", u.DisplayName)
	}
	if u.Role != UserRole_User {
		t.Errorf("expected Role 'user', got %q", u.Role)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if u.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNewUserWithProvidedID(t *testing.T) {
	id := "01JCUSTOM"
	u := NewUser(&id, authentication.AuthenticationProvider_Clerk, "clerk_xyz", "bob@example.com", "Bob", UserRole_User)

	if u.UserID != "01JCUSTOM" {
		t.Errorf("expected UserID '01JCUSTOM', got %q", u.UserID)
	}
}

func TestNewUserWithEmptyID(t *testing.T) {
	id := ""
	u := NewUser(&id, authentication.AuthenticationProvider_Clerk, "clerk_xyz", "bob@example.com", "Bob", UserRole_User)

	if u.UserID == "" {
		t.Fatal("expected UserID to be auto-generated when empty string provided")
	}
}

func TestUserKeys(t *testing.T) {
	u := &User{
		UserID:                 "01JTEST",
		AuthenticationProvider: authentication.AuthenticationProvider_Clerk,
		AuthenticationID:       "clerk_abc123",
	}

	if got := u.PK(); got != "USER#01JTEST" {
		t.Errorf("PK() = %q, want %q", got, "USER#01JTEST")
	}
	if got := u.SK(); got != "USER#01JTEST" {
		t.Errorf("SK() = %q, want %q", got, "USER#01JTEST")
	}

	gsis := u.GSIs()
	gsi1, ok := gsis[1]
	if !ok {
		t.Fatal("expected GSI1 to be present")
	}
	if gsi1.PK != "PROVIDER#CLERK" {
		t.Errorf("GSI1 PK = %q, want %q", gsi1.PK, "PROVIDER#CLERK")
	}
	if gsi1.SK != "AUTHENTICATION_ID#clerk_abc123" {
		t.Errorf("GSI1 SK = %q, want %q", gsi1.SK, "AUTHENTICATION_ID#clerk_abc123")
	}
}

func TestUserIsAdmin(t *testing.T) {
	u := &User{Role: UserRole_Admin}
	if !u.IsAdmin() {
		t.Error("expected IsAdmin() to return true for admin role")
	}

	u.Role = UserRole_User
	if u.IsAdmin() {
		t.Error("expected IsAdmin() to return false for user role")
	}

	u.Role = UserRole_User
	if u.IsAdmin() {
		t.Error("expected IsAdmin() to return false for empty role")
	}
}

func TestUserRegistration(t *testing.T) {
	model, err := Lookup("USER#01JTEST", "USER#01JTEST")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := model.(*User); !ok {
		t.Errorf("expected *User, got %T", model)
	}
}
