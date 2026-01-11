// Package router tests
package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/handlers"
	"gosveltekit/internal/models"
	"gosveltekit/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockAuthService implements service.AuthServiceInterface
type MockAuthService struct{}

func (m *MockAuthService) Login(username, password, ip, userAgent string) (*service.LoginResponse, error) {
	return &service.LoginResponse{
		SessionID: "mock-session-id",
		ExpiresAt: time.Now().Add(time.Hour),
		User: auth.UserData{
			ID:         "1",
			Identifier: username,
		},
	}, nil
}

func (m *MockAuthService) ValidateSession(sessionID string) (*auth.Session, *auth.UserData, error) {
	return &auth.Session{
			ID:        sessionID,
			UserID:    "1",
			ExpiresAt: time.Now().Add(time.Hour),
		}, &auth.UserData{
			ID:         "1",
			Identifier: "testuser",
			Role:       "user",
		}, nil
}

func (m *MockAuthService) Logout(sessionID string) error {
	return nil
}

func (m *MockAuthService) LogoutAll(userID string) error {
	return nil
}

func (m *MockAuthService) Register(username, email, password, displayName string) (*models.User, error) {
	return &models.User{}, nil
}

func (m *MockAuthService) RequestPasswordReset(email string) error {
	return nil
}

func (m *MockAuthService) ResetPassword(token, newPassword string) error {
	return nil
}

func NewMockAuthHandler() *handlers.AuthHandler {
	mockAuthService := &MockAuthService{}
	return handlers.NewAuthHandler(mockAuthService)
}

func NewMockAuthManager() *auth.AuthManager {
	// Create in-memory database for testing
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Session{})

	userAdapter := gormadapter.NewUserAdapter(db)
	sessionAdapter := gormadapter.NewSessionAdapter(db)

	return auth.NewAuthManager(userAdapter, sessionAdapter, auth.DefaultAuthConfig())
}

func TestSetupRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	mockAuthHandler := NewMockAuthHandler()
	mockAuthManager := NewMockAuthManager()
	router := SetupRouter(mockAuthHandler, mockAuthManager)

	// Test cases structure
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:           "Root endpoint",
			method:         "GET",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"message": "Hello GoSvelteKit"},
		},
		{
			name:           "Ping endpoint",
			method:         "GET",
			path:           "/ping",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"message": "pong"},
		},
		{
			name:           "Health endpoint",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"status": "ok"},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			for key, expectedValue := range tt.expectedBody {
				if actualValue := response[key]; actualValue != expectedValue {
					t.Errorf("Expected %s to be %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	mockAuthHandler := NewMockAuthHandler()
	mockAuthManager := NewMockAuthManager()
	router := SetupRouter(mockAuthHandler, mockAuthManager)

	// Test auth routes rate limiting
	t.Run("Auth routes rate limiting", func(t *testing.T) {
		path := "/auth/login"

		// Make multiple requests to trigger rate limiting
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", path, nil)
			req.RemoteAddr = "192.0.2.1:1234" // Set a consistent IP for testing
			router.ServeHTTP(w, req)

			if i < 3 {
				if w.Code == http.StatusTooManyRequests {
					t.Errorf("Request %d should not be rate limited", i+1)
				}
			} else {
				if w.Code != http.StatusTooManyRequests {
					t.Errorf("Request %d should be rate limited", i+1)
				}
			}
		}
	})
}

func TestProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	mockAuthHandler := NewMockAuthHandler()
	mockAuthManager := NewMockAuthManager()
	router := SetupRouter(mockAuthHandler, mockAuthManager)

	tests := []struct {
		name           string
		method         string
		path           string
		withAuth       bool
		sessionID      string
		expectedStatus int
	}{
		{
			name:           "Access protected route without auth",
			method:         "POST",
			path:           "/api/logout",
			withAuth:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Access admin route without auth",
			method:         "GET",
			path:           "/api/admin/dashboard",
			withAuth:       false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)

			if tt.withAuth {
				req.Header.Set("Authorization", "Bearer "+tt.sessionID)
			}

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
