// Package models defines the data models for the application.
package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	gorm.Model
	// Identity information
	Username     string `gorm:"unique;not null;index" json:"username"`
	Email        string `gorm:"unique;not null;index" json:"email"`
	DisplayName  string `gorm:"not null" json:"display_name"`
	PasswordHash string `gorm:"not null" json:"-"`

	// Profile information
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`

	// Account status
	Active        bool      `gorm:"default:true" json:"active"`
	EmailVerified bool      `gorm:"default:false" json:"email_verified"`
	LastLogin     time.Time `json:"last_login,omitempty"`
	LastActive    time.Time `json:"last_active,omitempty"`

	// Access control
	Role        string `gorm:"default:user" json:"role"`
	Permissions string `gorm:"type:text" json:"permissions,omitempty"` // JSON string of permissions

	// Password reset (kept separate from session management)
	ResetToken       string    `json:"-"`
	ResetTokenExpiry time.Time `json:"-"`
}
