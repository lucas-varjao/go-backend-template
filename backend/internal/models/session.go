package models

import (
	"time"
)

// Session represents an authentication session stored in the database
type Session struct {
	ID        string    `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UserAgent string    `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
	IP        string    `gorm:"type:varchar(45)" json:"ip,omitempty"` // Supports IPv6
}

// TableName specifies the table name for GORM
func (Session) TableName() string {
	return "sessions"
}
