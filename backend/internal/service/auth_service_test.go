// Package service tests
package service

import (
	"testing"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/email"
	"gosveltekit/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test helpers
func setupTest(t *testing.T) (*AuthService, *auth.AuthManager, *gormadapter.UserAdapter, *gormadapter.SessionAdapter, *email.MockEmailService, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Session{})
	require.NoError(t, err)

	userAdapter := gormadapter.NewUserAdapter(db)
	sessionAdapter := gormadapter.NewSessionAdapter(db)
	authConfig := auth.DefaultAuthConfig()
	authManager := auth.NewAuthManager(userAdapter, sessionAdapter, authConfig)
	mockEmailService := email.NewMockEmailService()
	authService := NewAuthService(authManager, userAdapter, mockEmailService)

	return authService, authManager, userAdapter, sessionAdapter, mockEmailService, db
}

func createTestUser(t *testing.T, db *gorm.DB) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		DisplayName:  "Test User",
		PasswordHash: string(hashedPassword),
		Active:       true,
		Role:         "user",
	}

	err = db.Create(user).Error
	require.NoError(t, err)

	return user
}

// Tests
func TestNewAuthService(t *testing.T) {
	authService, _, _, _, _, _ := setupTest(t)
	assert.NotNil(t, authService)
	assert.NotNil(t, authService.authManager)
	assert.NotNil(t, authService.userAdapter)
	assert.NotNil(t, authService.emailService)
}

func TestAuthService_Login_Success(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	user := createTestUser(t, db)

	response, err := authService.Login("testuser", "password123", "127.0.0.1", "test-agent")

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.SessionID)
	assert.NotZero(t, response.ExpiresAt)
	assert.Equal(t, user.Username, response.User.Identifier)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	_ = createTestUser(t, db)

	testCases := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "wrong password",
			username: "testuser",
			password: "wrongpass",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			password: "password123",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "empty credentials",
			username: "",
			password: "",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := authService.Login(tc.username, tc.password, "127.0.0.1", "test-agent")
			assert.Nil(t, response)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestAuthService_Login_AccountLocked(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	_ = createTestUser(t, db)

	// Attempt to login with wrong password 5 times
	for i := 0; i < 5; i++ {
		_, _ = authService.Login("testuser", "wrongpass", "127.0.0.1", "test-agent")
	}

	// Try one more time
	response, err := authService.Login("testuser", "wrongpass", "127.0.0.1", "test-agent")
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bloqueada")
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	user := createTestUser(t, db)

	// Deactivate user
	user.Active = false
	db.Save(user)

	response, err := authService.Login("testuser", "password123", "127.0.0.1", "test-agent")
	assert.Nil(t, response)
	assert.ErrorIs(t, err, ErrUserNotActive)
}

func TestAuthService_ValidateSession_Success(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	user := createTestUser(t, db)

	// First login to get a session
	loginResp, err := authService.Login("testuser", "password123", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Validate the session
	session, userData, err := authService.ValidateSession(loginResp.SessionID)

	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotNil(t, userData)
	assert.Equal(t, user.Username, userData.Identifier)
}

func TestAuthService_ValidateSession_Invalid(t *testing.T) {
	authService, _, _, _, _, _ := setupTest(t)

	session, userData, err := authService.ValidateSession("invalid-session-id")
	assert.Nil(t, session)
	assert.Nil(t, userData)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_Logout(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	_ = createTestUser(t, db)

	// First login to get a session
	loginResp, err := authService.Login("testuser", "password123", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Logout
	err = authService.Logout(loginResp.SessionID)
	require.NoError(t, err)

	// Verify session is invalid
	_, _, err = authService.ValidateSession(loginResp.SessionID)
	assert.Error(t, err)
}

func TestAuthService_Register_Success(t *testing.T) {
	authService, _, _, _, _, _ := setupTest(t)

	user, err := authService.Register("newuser", "new@example.com", "password123", "New User")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, "New User", user.DisplayName)
	assert.NotEmpty(t, user.PasswordHash)
	assert.True(t, user.Active)
}

func TestAuthService_Register_DuplicateUser(t *testing.T) {
	authService, _, _, _, _, db := setupTest(t)
	_ = createTestUser(t, db)

	// Try to register with same username
	user, err := authService.Register("testuser", "another@example.com", "password123", "Another User")
	assert.Nil(t, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")

	// Try to register with same email
	user, err = authService.Register("anotheruser", "test@example.com", "password123", "Another User")
	assert.Nil(t, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
}

func TestAuthService_RequestPasswordReset(t *testing.T) {
	authService, _, _, _, mockEmailService, db := setupTest(t)
	user := createTestUser(t, db)

	err := authService.RequestPasswordReset(user.Email)
	require.NoError(t, err)

	// Verify that reset token was set
	var updatedUser models.User
	err = db.First(&updatedUser, user.ID).Error
	require.NoError(t, err)
	assert.NotEmpty(t, updatedUser.ResetToken)
	assert.False(t, updatedUser.ResetTokenExpiry.IsZero())

	// Verify that email was sent
	sentEmails := mockEmailService.GetSentEmails()
	require.Len(t, sentEmails, 1)
	assert.Equal(t, user.Email, sentEmails[0].To)
	assert.Equal(t, user.Username, sentEmails[0].Username)
	assert.Equal(t, user.DisplayName, sentEmails[0].DisplayName)
	assert.NotEmpty(t, sentEmails[0].Token)
}
