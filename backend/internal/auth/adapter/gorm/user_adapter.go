// Package gorm provides GORM-based implementations of auth adapters
package gorm

import (
	"strconv"
	"time"

	"gosveltekit/internal/auth"
	"gosveltekit/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserAdapter implements auth.UserAdapter using GORM
type UserAdapter struct {
	db *gorm.DB
}

// NewUserAdapter creates a new GORM-based user adapter
func NewUserAdapter(db *gorm.DB) *UserAdapter {
	return &UserAdapter{db: db}
}

// FindUserByIdentifier looks up user by username or email
func (a *UserAdapter) FindUserByIdentifier(identifier string) (*auth.UserData, error) {
	var user models.User
	err := a.db.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrInvalidCredentials
		}
		return nil, err
	}
	return a.toUserData(&user), nil
}

// FindUserByID looks up user by ID
func (a *UserAdapter) FindUserByID(id string) (*auth.UserData, error) {
	userID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	var user models.User
	if err := a.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, auth.ErrInvalidCredentials
		}
		return nil, err
	}
	return a.toUserData(&user), nil
}

// ValidateCredentials validates username/email and password
func (a *UserAdapter) ValidateCredentials(identifier, password string) (*auth.UserData, error) {
	var user models.User
	err := a.db.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
	if err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	// Update last login time
	user.LastLogin = time.Now()
	a.db.Save(&user)

	return a.toUserData(&user), nil
}

// CreateUser creates a new user
func (a *UserAdapter) CreateUser(data auth.CreateUserInput) (*auth.UserData, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     data.Identifier,
		Email:        data.Email,
		DisplayName:  data.DisplayName,
		PasswordHash: string(hashedPassword),
		Active:       true,
		Role:         "user",
	}

	if err := a.db.Create(user).Error; err != nil {
		return nil, err
	}

	return a.toUserData(user), nil
}

// UpdatePassword updates the user's password
func (a *UserAdapter) UpdatePassword(userID string, newPassword string) error {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return a.db.Model(&models.User{}).Where("id = ?", id).Update("password_hash", string(hashedPassword)).Error
}

// GetUserModel returns the underlying GORM user model (for advanced queries)
func (a *UserAdapter) GetUserModel(userID string) (*models.User, error) {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := a.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds user by email (for password reset)
func (a *UserAdapter) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser saves changes to user model
func (a *UserAdapter) UpdateUser(user *models.User) error {
	return a.db.Save(user).Error
}

func (a *UserAdapter) toUserData(user *models.User) *auth.UserData {
	return &auth.UserData{
		ID:          strconv.FormatUint(uint64(user.ID), 10),
		Identifier:  user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Active:      user.Active,
		Attributes: map[string]any{
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"email_verified": user.EmailVerified,
			"last_login":     user.LastLogin,
		},
	}
}
