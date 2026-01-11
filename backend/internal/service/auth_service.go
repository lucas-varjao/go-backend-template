// Package service provides business logic services for the application.
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/email"
	"gosveltekit/internal/models"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrUserNotActive      = errors.New("usuário inativo")
	ErrInvalidToken       = errors.New("token inválido")
	ErrExpiredToken       = errors.New("token expirado")
)

// AuthServiceInterface defines the methods that an auth service must implement
type AuthServiceInterface interface {
	Login(username, password, ip, userAgent string) (*LoginResponse, error)
	ValidateSession(sessionID string) (*auth.Session, *auth.UserData, error)
	Logout(sessionID string) error
	LogoutAll(userID string) error
	Register(username, email, password, displayName string) (*models.User, error)
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
}

// AuthService handles authentication business logic
type AuthService struct {
	authManager  *auth.AuthManager
	userAdapter  *gormadapter.UserAdapter
	emailService email.EmailServiceInterface
}

// NewAuthService creates a new AuthService instance
func NewAuthService(
	authManager *auth.AuthManager,
	userAdapter *gormadapter.UserAdapter,
	emailService email.EmailServiceInterface,
) *AuthService {
	return &AuthService{
		authManager:  authManager,
		userAdapter:  userAdapter,
		emailService: emailService,
	}
}

// LoginResponse represents the response from a successful login
type LoginResponse struct {
	SessionID string        `json:"session_id"`
	ExpiresAt time.Time     `json:"expires_at"`
	User      auth.UserData `json:"user"`
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(username, password, ip, userAgent string) (*LoginResponse, error) {
	metadata := auth.SessionMetadata{
		UserAgent: userAgent,
		IP:        ip,
	}

	session, user, err := s.authManager.Login(username, password, metadata)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			return nil, ErrInvalidCredentials
		case errors.Is(err, auth.ErrUserNotActive):
			return nil, ErrUserNotActive
		case errors.Is(err, auth.ErrAccountLocked):
			return nil, errors.New("conta temporariamente bloqueada, tente novamente mais tarde")
		default:
			return nil, err
		}
	}

	return &LoginResponse{
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
		User:      *user,
	}, nil
}

// ValidateSession validates a session and returns user data
func (s *AuthService) ValidateSession(sessionID string) (*auth.Session, *auth.UserData, error) {
	session, user, err := s.authManager.ValidateSession(sessionID)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrSessionNotFound):
			return nil, nil, ErrInvalidToken
		case errors.Is(err, auth.ErrSessionExpired):
			return nil, nil, ErrExpiredToken
		case errors.Is(err, auth.ErrUserNotActive):
			return nil, nil, ErrUserNotActive
		default:
			return nil, nil, err
		}
	}
	return session, user, nil
}

// Logout invalidates a session
func (s *AuthService) Logout(sessionID string) error {
	return s.authManager.Logout(sessionID)
}

// LogoutAll invalidates all sessions for a user
func (s *AuthService) LogoutAll(userID string) error {
	return s.authManager.LogoutAll(userID)
}

// Register creates a new user account
func (s *AuthService) Register(username, email, password, displayName string) (*models.User, error) {
	// Check if username already exists
	if _, err := s.userAdapter.FindUserByIdentifier(username); err == nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	if _, err := s.userAdapter.FindByEmail(email); err == nil {
		return nil, errors.New("email already exists")
	}

	// Create user via adapter
	userData, err := s.userAdapter.CreateUser(auth.CreateUserInput{
		Identifier:  username,
		Email:       email,
		Password:    password,
		DisplayName: displayName,
	})
	if err != nil {
		return nil, err
	}

	// Get the actual User model for response
	user, err := s.userAdapter.GetUserModel(userData.ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// RequestPasswordReset initiates a password reset flow
func (s *AuthService) RequestPasswordReset(emailAddr string) error {
	user, err := s.userAdapter.FindByEmail(emailAddr)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Generate reset token
	tokenBytes := make([]byte, 32)
	_, err = s.generateSecureToken(tokenBytes)
	if err != nil {
		return err
	}

	plaintextToken := hex.EncodeToString(tokenBytes)
	hashedToken := s.hashToken(plaintextToken)
	expiresAt := time.Now().Add(1 * time.Hour)

	// Store hashed token
	user.ResetToken = hashedToken
	user.ResetTokenExpiry = expiresAt
	if err := s.userAdapter.UpdateUser(user); err != nil {
		return err
	}

	// Send email
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}

	if err := s.emailService.SendPasswordResetEmail(
		user.Email,
		plaintextToken,
		user.Username,
		displayName,
	); err != nil {
		fmt.Printf("Error sending password reset email: %v\n", err)
	}

	return nil
}

// ResetPassword resets a user's password using a reset token
func (s *AuthService) ResetPassword(tokenFromUser, newPassword string) error {
	// Hash the provided token and find matching user
	hashedToken := s.hashToken(tokenFromUser)

	// Find user with this reset token
	// This is a simplified implementation - in production you might want
	// to search by the hashed token directly
	users, err := s.findUsersWithResetTokens()
	if err != nil {
		return err
	}

	var matchedUser *models.User
	for _, user := range users {
		if time.Now().After(user.ResetTokenExpiry) {
			continue
		}
		if user.ResetToken == hashedToken {
			matchedUser = user
			break
		}
	}

	if matchedUser == nil {
		return ErrInvalidToken
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password and clear reset token
	matchedUser.PasswordHash = string(hashedPassword)
	matchedUser.ResetToken = ""
	matchedUser.ResetTokenExpiry = time.Time{}

	// Also invalidate all existing sessions for security
	userID := strconv.FormatUint(uint64(matchedUser.ID), 10)
	_ = s.authManager.LogoutAll(userID)

	return s.userAdapter.UpdateUser(matchedUser)
}

// Helper methods

func (s *AuthService) generateSecureToken(b []byte) (int, error) {
	return auth.GenerateRandomBytes(b)
}

func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) findUsersWithResetTokens() ([]*models.User, error) {
	// This method would need to be added to the userAdapter
	// For now, we'll use a workaround
	return nil, errors.New("not implemented - use direct DB query")
}

// ConvertToPublicUser strips sensitive fields from user
func ConvertToPublicUser(user *models.User) *models.User {
	user.PasswordHash = ""
	user.ResetToken = ""
	return user
}

// ParseUserID converts a string user ID to uint
func ParseUserID(id string) (uint, error) {
	parsed, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

// Helper to extract session ID from token string
func ExtractSessionID(token string) string {
	return strings.TrimPrefix(token, "Bearer ")
}
