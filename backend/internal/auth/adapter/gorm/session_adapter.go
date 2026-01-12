package gorm

import (
	"strconv"
	"time"

	"gosveltekit/internal/auth"
	"gosveltekit/internal/logger"
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
		logger.Error("Erro ao parsear userID para criar sessão", "error", err, "user_id", userID)
		return nil, err
	}

	// Generate session ID
	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		logger.Error("Erro ao gerar ID de sessão", "error", err, "user_id", userID)
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
		logger.Error("Erro ao criar sessão no banco de dados", "error", err, "user_id", userID, "session_id", sessionID)
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
		logger.Error("Erro ao buscar sessão no banco de dados", "error", err, "session_id", sessionID)
		return nil, err
	}

	return a.toAuthSession(&session), nil
}

// UpdateSessionExpiry updates the expiration time of a session
func (a *SessionAdapter) UpdateSessionExpiry(sessionID string, expiresAt time.Time) error {
	if err := a.db.Model(&models.Session{}).Where("id = ?", sessionID).Update("expires_at", expiresAt).Error; err != nil {
		logger.Error("Erro ao atualizar expiração da sessão", "error", err, "session_id", sessionID)
		return err
	}
	return nil
}

// DeleteSession removes a session
func (a *SessionAdapter) DeleteSession(sessionID string) error {
	if err := a.db.Where("id = ?", sessionID).Delete(&models.Session{}).Error; err != nil {
		logger.Error("Erro ao deletar sessão", "error", err, "session_id", sessionID)
		return err
	}
	return nil
}

// DeleteUserSessions removes all sessions for a user
func (a *SessionAdapter) DeleteUserSessions(userID string) error {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		logger.Error("Erro ao parsear userID para deletar sessões", "error", err, "user_id", userID)
		return err
	}
	if err := a.db.Where("user_id = ?", uid).Delete(&models.Session{}).Error; err != nil {
		logger.Error("Erro ao deletar sessões do usuário", "error", err, "user_id", userID)
		return err
	}
	return nil
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
