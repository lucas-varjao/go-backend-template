// Package repository provides data access layer for the application.
package repository

import (
	"gosveltekit/internal/models"

	"gorm.io/gorm"
)

// UserRepository provides access to users in the database
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID finds a user by their ID
func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by their email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByResetToken finds a user by their password reset token
func (r *UserRepository) FindByResetToken(token string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("reset_token = ?", token).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by their username
func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUsersWithResetTokens finds all users that currently have reset tokens
func (r *UserRepository) FindUsersWithResetTokens() ([]*models.User, error) {
	var users []*models.User
	// Find all users where reset_token is not empty
	if err := r.db.Where("reset_token <> ''").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// Update saves changes to a user in the database
func (r *UserRepository) Update(user *models.User) error {
	// Validate required fields
	if user.Email == "" {
		return gorm.ErrInvalidField
	}
	if user.Username == "" {
		return gorm.ErrInvalidField
	}
	if user.DisplayName == "" {
		return gorm.ErrInvalidField
	}
	return r.db.Save(user).Error
}
