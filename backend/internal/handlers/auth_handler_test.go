// Package handlers tests
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gosveltekit/internal/auth"
	"gosveltekit/internal/models"
	"gosveltekit/internal/service"

	"github.com/gin-gonic/gin"
)

// MockAuthService implements the service.AuthServiceInterface interface
type MockAuthService struct {
	LoginFunc                func(username, password, ip, userAgent string) (*service.LoginResponse, error)
	ValidateSessionFunc      func(sessionID string) (*auth.Session, *auth.UserData, error)
	LogoutFunc               func(sessionID string) error
	LogoutAllFunc            func(userID string) error
	RegisterFunc             func(username, email, password, displayName string) (*models.User, error)
	RequestPasswordResetFunc func(email string) error
	ResetPasswordFunc        func(token, newPassword string) error
}

func (m *MockAuthService) Login(username, password, ip, userAgent string) (*service.LoginResponse, error) {
	return m.LoginFunc(username, password, ip, userAgent)
}

func (m *MockAuthService) ValidateSession(sessionID string) (*auth.Session, *auth.UserData, error) {
	return m.ValidateSessionFunc(sessionID)
}

func (m *MockAuthService) Logout(sessionID string) error {
	return m.LogoutFunc(sessionID)
}

func (m *MockAuthService) LogoutAll(userID string) error {
	return m.LogoutAllFunc(userID)
}

func (m *MockAuthService) Register(username, email, password, displayName string) (*models.User, error) {
	return m.RegisterFunc(username, email, password, displayName)
}

func (m *MockAuthService) RequestPasswordReset(email string) error {
	return m.RequestPasswordResetFunc(email)
}

func (m *MockAuthService) ResetPassword(token, newPassword string) error {
	return m.ResetPasswordFunc(token, newPassword)
}

func setupTestRouter() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestNewAuthHandler(t *testing.T) {
	var mockService service.AuthServiceInterface = &MockAuthService{}
	handler := NewAuthHandler(mockService)

	if handler == nil {
		t.Fatal("NewAuthHandler returned nil")
		return
	}
	if handler.authService == nil {
		t.Error("authService was not set")
		return
	}
	if handler.authService != mockService {
		t.Error("authService not properly set to the provided mock")
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		request        LoginRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Successful login",
			request: LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.LoginFunc = func(username, password, ip, userAgent string) (*service.LoginResponse, error) {
					return &service.LoginResponse{
						SessionID: "test-session-id",
						ExpiresAt: time.Now().Add(time.Hour),
						User: auth.UserData{
							ID:         "1",
							Identifier: "testuser",
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"session_id": "test-session-id",
			},
		},
		{
			name: "Invalid credentials",
			request: LoginRequest{
				Username: "testuser",
				Password: "wrongpass",
			},
			setupMock: func(m *MockAuthService) {
				m.LoginFunc = func(username, password, ip, userAgent string) (*service.LoginResponse, error) {
					return nil, service.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "credenciais inválidas",
			},
		},
		{
			name: "Inactive user",
			request: LoginRequest{
				Username: "inactive",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.LoginFunc = func(username, password, ip, userAgent string) (*service.LoginResponse, error) {
					return nil, service.ErrUserNotActive
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "usuário inativo",
			},
		},
		{
			name: "Account locked",
			request: LoginRequest{
				Username: "locked",
				Password: "password123",
			},
			setupMock: func(m *MockAuthService) {
				m.LoginFunc = func(username, password, ip, userAgent string) (*service.LoginResponse, error) {
					return nil, errors.New("conta temporariamente bloqueada, tente novamente mais tarde")
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "conta temporariamente bloqueada, tente novamente mais tarde",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestRouter()
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			var authService service.AuthServiceInterface = mockService
			handler := NewAuthHandler(authService)

			// Setup request
			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			// Call handler
			handler.Login(c)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Check expected fields
			for key, expectedValue := range tt.expectedBody {
				if actualValue, exists := response[key]; !exists {
					t.Errorf("expected response to contain %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Successful logout",
			setupContext: func(c *gin.Context) {
				c.Set("userID", "1")
				c.Set("sessionID", "valid-session")
			},
			setupMock: func(m *MockAuthService) {
				m.LogoutFunc = func(sessionID string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "logout realizado com sucesso",
			},
		},
		{
			name: "Unauthorized - no session ID",
			setupContext: func(c *gin.Context) {
				// Don't set sessionID
			},
			setupMock: func(m *MockAuthService) {
				m.LogoutFunc = func(sessionID string) error {
					return nil
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "não autenticado",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestRouter()
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			var authService service.AuthServiceInterface = mockService
			handler := NewAuthHandler(authService)

			// Setup context
			tt.setupContext(c)

			// Call handler
			handler.Logout(c)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			for key, expectedValue := range tt.expectedBody {
				if actualValue, exists := response[key]; !exists {
					t.Errorf("expected response to contain %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		request        RegistrationRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Successful registration",
			request: RegistrationRequest{
				Username:    "newuser",
				Email:       "new@example.com",
				Password:    "Padasdasdasdd123!",
				DisplayName: "New User",
			},
			setupMock: func(m *MockAuthService) {
				m.RegisterFunc = func(username, email, password, displayName string) (*models.User, error) {
					return &models.User{
						Username:    username,
						Email:       email,
						DisplayName: displayName,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"username": "newuser",
				"email":    "new@example.com",
			},
		},
		{
			name: "Username already exists",
			request: RegistrationRequest{
				Username:    "existinguser",
				Email:       "new@example.com",
				Password:    "@aSdasdsads123!",
				DisplayName: "New User",
			},
			setupMock: func(m *MockAuthService) {
				m.RegisterFunc = func(username, email, password, displayName string) (*models.User, error) {
					return nil, errors.New("username already exists")
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "username already exists",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestRouter()
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			var authService service.AuthServiceInterface = mockService
			handler := NewAuthHandler(authService)

			// Setup request
			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			// Call handler
			handler.Register(c)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			for key, expectedValue := range tt.expectedBody {
				if actualValue, exists := response[key]; !exists {
					t.Errorf("expected response to contain %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestAuthHandler_RequestPasswordReset(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   interface{}
		checkBody      func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "Successful password reset request",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			setupMock: func(m *MockAuthService) {
				m.RequestPasswordResetFunc = func(email string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "se o email existir, um link de recuperação será enviado",
			},
		},
		{
			name: "Invalid email format",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			setupMock: func(m *MockAuthService) {
				m.RequestPasswordResetFunc = func(email string) error {
					return errors.New("should not be called")
				}
			},
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				if !contains(body["error"].(string), "validation") && !contains(body["error"].(string), "email") {
					t.Errorf("expected error message to mention email validation, got: %v", body["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestRouter()
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			var authService service.AuthServiceInterface = mockService
			handler := NewAuthHandler(authService)

			// Setup request
			jsonData, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/auth/password-reset-request", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			// Call handler
			handler.RequestPasswordReset(c)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectedBody != nil {
				expectedBody := tt.expectedBody.(map[string]interface{})
				for key, expectedValue := range expectedBody {
					if actualValue, exists := response[key]; !exists {
						t.Errorf("expected response to contain %s", key)
					} else if actualValue != expectedValue {
						t.Errorf("expected %s to be %v, got %v", key, expectedValue, actualValue)
					}
				}
			}

			if tt.checkBody != nil {
				tt.checkBody(t, response)
			}
		})
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return s != "" && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	tests := []struct {
		name           string
		request        PasswordResetRequest
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Successful password reset",
			request: PasswordResetRequest{
				Token:           "valid-token",
				NewPassword:     "NewPgdfgdfgd123!",
				ConfirmPassword: "NewPgdfgdfgd123!",
			},
			setupMock: func(m *MockAuthService) {
				m.ResetPasswordFunc = func(token, newPassword string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "senha redefinida com sucesso",
			},
		},
		{
			name: "Invalid token",
			request: PasswordResetRequest{
				Token:           "invalid-token",
				NewPassword:     "NewPgdfgdfgd123!",
				ConfirmPassword: "NewPgdfgdfgd123!",
			},
			setupMock: func(m *MockAuthService) {
				m.ResetPasswordFunc = func(token, newPassword string) error {
					return service.ErrInvalidToken
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "token inválido",
			},
		},
		{
			name: "Expired token",
			request: PasswordResetRequest{
				Token:           "expired-token",
				NewPassword:     "NewPgdfgdfgd123!",
				ConfirmPassword: "NewPgdfgdfgd123!",
			},
			setupMock: func(m *MockAuthService) {
				m.ResetPasswordFunc = func(token, newPassword string) error {
					return service.ErrExpiredToken
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "token expirado",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestRouter()
			mockService := &MockAuthService{}
			tt.setupMock(mockService)

			var authService service.AuthServiceInterface = mockService
			handler := NewAuthHandler(authService)

			// Setup request
			jsonData, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest(http.MethodPost, "/auth/password-reset", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			// Call handler
			handler.ResetPassword(c)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			for key, expectedValue := range tt.expectedBody {
				if actualValue, exists := response[key]; !exists {
					t.Errorf("expected response to contain %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}
