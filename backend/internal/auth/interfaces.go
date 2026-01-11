// Package auth provides a pluggable authentication system inspired by Lucia Auth.
//
// The package uses an adapter pattern that allows connecting to any user database.
// Key components:
//   - UserAdapter: Interface for user lookup and credential validation
//   - SessionAdapter: Interface for session management
//   - AuthManager: Central manager that coordinates authentication flow
package auth

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user not active")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

// UserData represents generic user data (database-agnostic)
type UserData struct {
	ID          string         `json:"id"`
	Identifier  string         `json:"identifier"` // username, email, etc
	DisplayName string         `json:"display_name"`
	Email       string         `json:"email"`
	Role        string         `json:"role"`
	Active      bool           `json:"active"`
	Attributes  map[string]any `json:"attributes,omitempty"` // extra fields
}

// Session represents an authentication session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UserAgent string    `json:"user_agent,omitempty"`
	IP        string    `json:"ip,omitempty"`
	Fresh     bool      `json:"fresh"` // true if just created or refreshed
}

// SessionMetadata contains metadata for session creation
type SessionMetadata struct {
	UserAgent string
	IP        string
}

// CreateUserInput contains data for creating a new user
type CreateUserInput struct {
	Identifier  string
	Email       string
	Password    string
	DisplayName string
	Attributes  map[string]any
}

// UserAdapter is the interface that any user database must implement
type UserAdapter interface {
	// FindUserByIdentifier looks up user by identifier (username, email, etc)
	FindUserByIdentifier(identifier string) (*UserData, error)

	// FindUserByID looks up user by ID
	FindUserByID(id string) (*UserData, error)

	// ValidateCredentials validates credentials and returns user if valid
	ValidateCredentials(identifier, password string) (*UserData, error)

	// CreateUser creates a new user (optional for legacy systems)
	CreateUser(data CreateUserInput) (*UserData, error)

	// UpdatePassword updates the password (optional for legacy systems)
	UpdatePassword(userID string, newPassword string) error
}

// SessionAdapter manages authentication sessions
type SessionAdapter interface {
	// CreateSession creates a new session for the user
	CreateSession(userID string, expiresAt time.Time, metadata SessionMetadata) (*Session, error)

	// GetSession retrieves a session by ID
	GetSession(sessionID string) (*Session, error)

	// UpdateSessionExpiry updates session expiration time
	UpdateSessionExpiry(sessionID string, expiresAt time.Time) error

	// DeleteSession removes a session (logout)
	DeleteSession(sessionID string) error

	// DeleteUserSessions removes all sessions for a user
	DeleteUserSessions(userID string) error

	// DeleteExpiredSessions cleans up expired sessions
	DeleteExpiredSessions() error
}

// PasswordResetAdapter optional interface for password reset functionality
type PasswordResetAdapter interface {
	// SetResetToken stores a password reset token for a user
	SetResetToken(userID string, hashedToken string, expiresAt time.Time) error

	// GetUserByResetToken finds user by reset token hash
	GetUserByResetToken(hashedToken string) (*UserData, error)

	// ClearResetToken clears the reset token after use
	ClearResetToken(userID string) error
}
