// Package handlers provides HTTP request handlers for the API.
package handlers

import (
	"net/http"

	"gosveltekit/internal/auth"
	"gosveltekit/internal/middleware"
	"gosveltekit/internal/service"
	"gosveltekit/internal/validation"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService service.AuthServiceInterface
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(authService service.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegistrationRequest represents the registration request body
type RegistrationRequest struct {
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
}

// PasswordResetRequest represents the password reset request body
type PasswordResetRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// Login handles user authentication with input validation
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input data before attempting login
	if err := validation.ValidateLoginRequest(req.Username, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get client IP and user agent
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	response, err := h.authService.Login(req.Username, req.Password, ip, userAgent)
	if err != nil {
		status := http.StatusUnauthorized
		message := "credenciais inválidas"

		switch {
		case err == service.ErrUserNotActive:
			message = "usuário inativo"
		case err.Error() == "conta temporariamente bloqueada, tente novamente mais tarde":
			message = err.Error()
		}

		c.JSON(status, gin.H{"error": message})
		return
	}

	// Set session cookie
	c.SetCookie(
		middleware.SessionCookieName,
		response.SessionID,
		30*24*60*60, // 30 days
		"/",
		"",
		true, // secure
		true, // httpOnly
	)

	c.JSON(http.StatusOK, response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, exists := c.Get("sessionID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}

	if err := h.authService.Logout(sessionID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao fazer logout"})
		return
	}

	// Clear session cookie
	middleware.ClearSessionCookie(c)

	c.JSON(http.StatusOK, gin.H{"message": "logout realizado com sucesso"})
}

// Register handles new user registration with comprehensive validation
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate all registration data
	if err := validation.ValidateRegistrationRequest(
		req.Username,
		req.Email,
		req.Password,
		req.DisplayName,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Forward to service layer
	user, err := h.authService.Register(req.Username, req.Email, req.Password, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Strip sensitive data
	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

// RequestPasswordReset handles password reset requests
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate email
	if err := validation.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		if err.Error() == "invalid email format" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Don't reveal if email exists for security reasons
		c.JSON(http.StatusOK, gin.H{"message": "se o email existir, um link de recuperação será enviado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "se o email existir, um link de recuperação será enviado"})
}

// ResetPassword handles password reset with token validation
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password reset request
	if err := validation.ValidatePasswordReset(req.Token, req.NewPassword, req.ConfirmPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		status := http.StatusBadRequest
		message := "falha ao redefinir senha"

		switch {
		case err == service.ErrInvalidToken:
			message = "token inválido"
		case err == service.ErrExpiredToken:
			message = "token expirado"
		}

		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "senha redefinida com sucesso"})
}

// GetCurrentUser returns the currently authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}

	c.JSON(http.StatusOK, user.(*auth.UserData))
}
