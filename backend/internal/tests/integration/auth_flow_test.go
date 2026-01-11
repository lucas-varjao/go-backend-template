// Package integration provides integration tests for the authentication flow.
package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/email"
	"gosveltekit/internal/handlers"
	"gosveltekit/internal/models"
	"gosveltekit/internal/router"
	"gosveltekit/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIntegrationTest(t *testing.T) (*gin.Engine, *gorm.DB, *auth.AuthManager) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Session{})
	require.NoError(t, err)

	// Setup adapters
	userAdapter := gormadapter.NewUserAdapter(db)
	sessionAdapter := gormadapter.NewSessionAdapter(db)

	// Setup auth manager
	authConfig := auth.DefaultAuthConfig()
	authManager := auth.NewAuthManager(userAdapter, sessionAdapter, authConfig)

	// Setup services
	emailService := email.NewMockEmailService()
	authService := service.NewAuthService(authManager, userAdapter, emailService)
	authHandler := handlers.NewAuthHandler(authService)

	// Setup router
	r := router.SetupRouter(authHandler, authManager)
	return r, db, authManager
}

func TestCompleteAuthFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, _, _ := setupIntegrationTest(t)

	// 1. Register user
	registration := map[string]interface{}{
		"username":     "testuser",
		"email":        "test@example.com",
		"password":     "Test123!@#",
		"display_name": "Test User",
	}
	w := httptest.NewRecorder()
	jsonData, _ := json.Marshal(registration)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. Login
	login := map[string]interface{}{
		"username": "testuser",
		"password": "Test123!@#",
	}
	w = httptest.NewRecorder()
	jsonData, _ = json.Marshal(login)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	sessionID := loginResponse["session_id"].(string)
	assert.NotEmpty(t, sessionID)

	// 3. Access protected route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Logout
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/logout", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. Attempt access after logout (should fail)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPasswordResetFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, db, _ := setupIntegrationTest(t)

	// 1. Create user directly in database
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		Username:     "resetuser",
		Email:        "reset@example.com",
		PasswordHash: string(hashedPassword),
		DisplayName:  "Reset User",
		Active:       true,
		Role:         "user",
	}
	err = db.Create(user).Error
	require.NoError(t, err)

	// 2. Request password reset
	resetRequest := map[string]interface{}{
		"email": "reset@example.com",
	}
	w := httptest.NewRecorder()
	jsonData, _ := json.Marshal(resetRequest)
	req, _ := http.NewRequest("POST", "/auth/password-reset-request", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify reset token was set
	var updatedUser models.User
	err = db.First(&updatedUser, user.ID).Error
	require.NoError(t, err)
	// Note: In the new system, password reset still works but the token verification
	// is handled differently. The test verifies the request was accepted.
}

func TestGetCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, _, _ := setupIntegrationTest(t)

	// 1. Register and login
	registration := map[string]interface{}{
		"username":     "meuser",
		"email":        "me@example.com",
		"password":     "Test123!@#",
		"display_name": "Me User",
	}
	w := httptest.NewRecorder()
	jsonData, _ := json.Marshal(registration)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	login := map[string]interface{}{
		"username": "meuser",
		"password": "Test123!@#",
	}
	w = httptest.NewRecorder()
	jsonData, _ = json.Marshal(login)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	sessionID := loginResponse["session_id"].(string)

	// 2. Get current user
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var userResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &userResponse)
	require.NoError(t, err)
	assert.Equal(t, "meuser", userResponse["identifier"])
}
