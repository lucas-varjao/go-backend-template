package gorm

import (
	"strconv"
	"time"

	"gosveltekit/internal/auth"
	"gosveltekit/internal/models"

	"gorm.io/gorm"
)

// SessionAdapter implements auth.SessionAdapter using GORM
type SessionAdapter struct {
	db *gorm.DB
}

// NewSessionAdapter creates a new GORM-based session adapter
func NewSessionAdapter(db *gorm.DB) *SessionAdapter {
	return &SessionAdapter{db: db}
}

// CreateSession creates a new session for a user
func (a *SessionAdapter) CreateSession(userID string, expiresAt time.Time, metadata auth.SessionMetadata) (*auth.Session, error) {
	// Parse userID as uint for GORM model
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, err
	}

	// Generate session ID
	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    uint(uid),
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		UserAgent: metadata.UserAgent,
		IP:        metadata.IP,
	}

	if err := a.db.Create(session).Error; err != nil {
		return nil, err
	}

	return a.toAuthSession(session), nil
}

// GetSession retrieves a session by ID
func (a *SessionAdapter) GetSession(sessionID string) (*auth.Session, error) {
	var session models.Session
	if err := a.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrSessionNotFound
		}
		return nil, err
	}

	return a.toAuthSession(&session), nil
}

// UpdateSessionExpiry updates the expiration time of a session
func (a *SessionAdapter) UpdateSessionExpiry(sessionID string, expiresAt time.Time) error {
	return a.db.Model(&models.Session{}).Where("id = ?", sessionID).Update("expires_at", expiresAt).Error
}

// DeleteSession removes a session
func (a *SessionAdapter) DeleteSession(sessionID string) error {
	return a.db.Where("id = ?", sessionID).Delete(&models.Session{}).Error
}

// DeleteUserSessions removes all sessions for a user
func (a *SessionAdapter) DeleteUserSessions(userID string) error {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return err
	}
	return a.db.Where("user_id = ?", uid).Delete(&models.Session{}).Error
}

// DeleteExpiredSessions cleans up expired sessions
func (a *SessionAdapter) DeleteExpiredSessions() error {
	return a.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{}).Error
}

func (a *SessionAdapter) toAuthSession(session *models.Session) *auth.Session {
	return &auth.Session{
		ID:        session.ID,
		UserID:    strconv.FormatUint(uint64(session.UserID), 10),
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
		UserAgent: session.UserAgent,
		IP:        session.IP,
	}
}
