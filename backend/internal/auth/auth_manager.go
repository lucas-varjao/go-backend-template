package auth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// AuthConfig holds configuration for the auth manager
type AuthConfig struct {
	SessionDuration   time.Duration // Default: 30 days
	RefreshThreshold  time.Duration // Refresh if less than this remaining (default: 15 days)
	MaxFailedAttempts int           // Max failed login attempts before lockout
	LockoutDuration   time.Duration // How long to lock account after max attempts
}

// DefaultAuthConfig returns sensible defaults
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		SessionDuration:   30 * 24 * time.Hour, // 30 days
		RefreshThreshold:  15 * 24 * time.Hour, // 15 days
		MaxFailedAttempts: 5,
		LockoutDuration:   30 * time.Minute,
	}
}

// AuthManager is the central authentication coordinator
type AuthManager struct {
	userAdapter    UserAdapter
	sessionAdapter SessionAdapter
	config         *AuthConfig

	// Rate limiting for failed attempts
	failedAttempts      map[string]failedAttemptInfo
	failedAttemptsMutex sync.RWMutex
}

type failedAttemptInfo struct {
	count    int
	lastTry  time.Time
	lockedAt time.Time
	isLocked bool
}

// NewAuthManager creates a new AuthManager instance
func NewAuthManager(userAdapter UserAdapter, sessionAdapter SessionAdapter, config *AuthConfig) *AuthManager {
	if config == nil {
		config = DefaultAuthConfig()
	}
	return &AuthManager{
		userAdapter:    userAdapter,
		sessionAdapter: sessionAdapter,
		config:         config,
		failedAttempts: make(map[string]failedAttemptInfo),
	}
}

// Login authenticates a user and creates a session
func (m *AuthManager) Login(identifier, password string, metadata SessionMetadata) (*Session, *UserData, error) {
	// Check if account is locked
	if m.isAccountLocked(identifier) {
		return nil, nil, ErrAccountLocked
	}

	// Validate credentials
	user, err := m.userAdapter.ValidateCredentials(identifier, password)
	if err != nil {
		m.recordFailedAttempt(identifier)
		return nil, nil, err
	}

	// Check if user is active
	if !user.Active {
		return nil, nil, ErrUserNotActive
	}

	// Clear failed attempts on successful login
	m.clearFailedAttempts(identifier)

	// Create session
	expiresAt := time.Now().Add(m.config.SessionDuration)
	session, err := m.sessionAdapter.CreateSession(user.ID, expiresAt, metadata)
	if err != nil {
		return nil, nil, err
	}

	session.Fresh = true
	return session, user, nil
}

// ValidateSession validates a session and returns user data
func (m *AuthManager) ValidateSession(sessionID string) (*Session, *UserData, error) {
	session, err := m.sessionAdapter.GetSession(sessionID)
	if err != nil {
		return nil, nil, ErrSessionNotFound
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		_ = m.sessionAdapter.DeleteSession(sessionID)
		return nil, nil, ErrSessionExpired
	}

	// Get user data
	user, err := m.userAdapter.FindUserByID(session.UserID)
	if err != nil {
		return nil, nil, err
	}

	// Check if user is still active
	if !user.Active {
		return nil, nil, ErrUserNotActive
	}

	// Refresh session if needed
	session.Fresh = false
	timeRemaining := time.Until(session.ExpiresAt)
	if timeRemaining < m.config.RefreshThreshold {
		newExpiresAt := time.Now().Add(m.config.SessionDuration)
		if err := m.sessionAdapter.UpdateSessionExpiry(sessionID, newExpiresAt); err == nil {
			session.ExpiresAt = newExpiresAt
			session.Fresh = true
		}
	}

	return session, user, nil
}

// Logout invalidates a session
func (m *AuthManager) Logout(sessionID string) error {
	return m.sessionAdapter.DeleteSession(sessionID)
}

// LogoutAll invalidates all sessions for a user
func (m *AuthManager) LogoutAll(userID string) error {
	return m.sessionAdapter.DeleteUserSessions(userID)
}

// GetUserAdapter returns the user adapter (useful for registration, etc)
func (m *AuthManager) GetUserAdapter() UserAdapter {
	return m.userAdapter
}

// GetSessionAdapter returns the session adapter
func (m *AuthManager) GetSessionAdapter() SessionAdapter {
	return m.sessionAdapter
}

// GenerateSessionID generates a cryptographically secure session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateRandomBytes fills a byte slice with cryptographically secure random bytes
func GenerateRandomBytes(b []byte) (int, error) {
	return rand.Read(b)
}

// --- Rate limiting helpers ---

func (m *AuthManager) isAccountLocked(identifier string) bool {
	m.failedAttemptsMutex.RLock()
	defer m.failedAttemptsMutex.RUnlock()

	info, exists := m.failedAttempts[identifier]
	if !exists {
		return false
	}

	if !info.isLocked {
		return false
	}

	// Check if lockout has expired
	if time.Since(info.lockedAt) > m.config.LockoutDuration {
		return false
	}

	return true
}

func (m *AuthManager) recordFailedAttempt(identifier string) {
	m.failedAttemptsMutex.Lock()
	defer m.failedAttemptsMutex.Unlock()

	info := m.failedAttempts[identifier]
	info.count++
	info.lastTry = time.Now()

	if info.count >= m.config.MaxFailedAttempts {
		info.isLocked = true
		info.lockedAt = time.Now()
	}

	m.failedAttempts[identifier] = info
}

func (m *AuthManager) clearFailedAttempts(identifier string) {
	m.failedAttemptsMutex.Lock()
	defer m.failedAttemptsMutex.Unlock()
	delete(m.failedAttempts, identifier)
}

// ErrAccountLocked is returned when an account is temporarily locked
var ErrAccountLocked = errorString("account temporarily locked")

type errorString string

func (e errorString) Error() string { return string(e) }
