// Package middleware provides HTTP middleware for the Gin router.
package middleware

import (
	"net/http"
	"strings"

	"gosveltekit/internal/auth"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "session_id"
	// SessionHeaderName is the name of the session header (for API clients)
	SessionHeaderName = "X-Session-ID"
)

// AuthMiddleware creates a Gin middleware for session-based authentication.
//
// It looks for a session ID in either:
// 1. The Authorization header (format: "Bearer {session_id}")
// 2. The X-Session-ID header
// 3. A cookie named "session_id"
//
// If validation succeeds, it adds user info to the request context.
func AuthMiddleware(authManager *auth.AuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := extractSessionID(c)
		if sessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "autorização necessária"})
			return
		}

		session, user, err := authManager.ValidateSession(sessionID)
		if err != nil {
			status := http.StatusUnauthorized
			message := "sessão inválida"

			switch {
			case err == auth.ErrSessionExpired:
				message = "sessão expirada"
			case err == auth.ErrSessionNotFound:
				message = "sessão não encontrada"
			case err == auth.ErrUserNotActive:
				message = "usuário inativo"
			}

			c.AbortWithStatusJSON(status, gin.H{"error": message})
			return
		}

		// Store user info in context
		c.Set("userID", user.ID)
		c.Set("role", user.Role)
		c.Set("user", user)
		c.Set("session", session)
		c.Set("sessionID", sessionID)

		// If session was refreshed, update the cookie
		if session.Fresh && c.Request.Method != http.MethodOptions {
			setSessionCookie(c, sessionID, session.ExpiresAt)
		}

		c.Next()
	}
}

// RoleMiddleware creates a middleware to verify user roles.
//
// It expects the user's role to be set in the context by AuthMiddleware.
func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
			return
		}

		for _, role := range roles {
			if role == userRole {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acesso negado"})
	}
}

// extractSessionID extracts the session ID from the request.
// Priority: Authorization header > X-Session-ID header > Cookie
func extractSessionID(c *gin.Context) string {
	// Try Authorization header first (for API clients)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Try X-Session-ID header
	if sessionID := c.GetHeader(SessionHeaderName); sessionID != "" {
		return sessionID
	}

	// Try cookie
	if cookie, err := c.Cookie(SessionCookieName); err == nil {
		return cookie
	}

	return ""
}

// setSessionCookie sets the session cookie in the response
func setSessionCookie(c *gin.Context, sessionID string, expiresAt interface{}) {
	// Calculate max age in seconds
	maxAge := 30 * 24 * 60 * 60 // 30 days default

	c.SetCookie(
		SessionCookieName,
		sessionID,
		maxAge,
		"/",
		"",   // domain - empty means current domain
		true, // secure - only send over HTTPS
		true, // httpOnly - not accessible via JavaScript
	)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(c *gin.Context) {
	c.SetCookie(
		SessionCookieName,
		"",
		-1, // negative max age deletes the cookie
		"/",
		"",
		true,
		true,
	)
}
