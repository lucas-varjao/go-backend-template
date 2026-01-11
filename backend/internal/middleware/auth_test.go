// Package middleware tests
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gosveltekit/internal/auth"
	gormadapter "gosveltekit/internal/auth/adapter/gorm"
	"gosveltekit/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Setup test environment
func init() {
	gin.SetMode(gin.TestMode)
}

// createTestAuthManager creates a test AuthManager with in-memory database
func createTestAuthManager() (*auth.AuthManager, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.User{}, &models.Session{})

	userAdapter := gormadapter.NewUserAdapter(db)
	sessionAdapter := gormadapter.NewSessionAdapter(db)

	return auth.NewAuthManager(userAdapter, sessionAdapter, auth.DefaultAuthConfig()), db
}

// Test cases for AuthMiddleware
func TestAuthMiddleware(t *testing.T) {
	t.Run("Missing Authorization Header", func(t *testing.T) {
		authManager, _ := createTestAuthManager()

		r := gin.New()
		r.Use(AuthMiddleware(authManager))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "autorização necessária")
	})

	t.Run("Invalid Session ID", func(t *testing.T) {
		authManager, _ := createTestAuthManager()

		r := gin.New()
		r.Use(AuthMiddleware(authManager))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-session-id")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "sessão não encontrada")
	})

	t.Run("Valid Session via Header", func(t *testing.T) {
		authManager, db := createTestAuthManager()

		// Create a valid session directly in the database
		session := &models.Session{
			ID:        "valid-session-id",
			UserID:    1,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}
		db.Create(session)

		// Create a user
		user := &models.User{
			Username:     "testuser",
			Email:        "test@example.com",
			DisplayName:  "Test User",
			PasswordHash: "hash",
			Active:       true,
			Role:         "user",
		}
		db.Create(user)

		r := gin.New()
		r.Use(AuthMiddleware(authManager))
		r.GET("/test", func(c *gin.Context) {
			userID, _ := c.Get("userID")
			role, _ := c.Get("role")
			c.JSON(http.StatusOK, gin.H{"userID": userID, "role": role})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-session-id")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Valid Session via X-Session-ID Header", func(t *testing.T) {
		authManager, db := createTestAuthManager()

		session := &models.Session{
			ID:        "header-session-id",
			UserID:    1,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}
		db.Create(session)

		user := &models.User{
			Username:     "testuser2",
			Email:        "test2@example.com",
			DisplayName:  "Test User 2",
			PasswordHash: "hash",
			Active:       true,
		}
		db.Create(user)

		r := gin.New()
		r.Use(AuthMiddleware(authManager))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(SessionHeaderName, "header-session-id")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test cases for RoleMiddleware
func TestRoleMiddleware(t *testing.T) {
	t.Run("No Role in Context", func(t *testing.T) {
		r := gin.New()
		r.Use(RoleMiddleware("admin"))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "usuário não autenticado")
	})

	t.Run("Role Matches Required", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("role", "admin")
			c.Next()
		})
		r.Use(RoleMiddleware("admin", "manager"))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("One of Multiple Roles Matches", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("role", "user")
			c.Next()
		})
		r.Use(RoleMiddleware("admin", "user", "manager"))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Role Doesn't Match Required", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("role", "user")
			c.Next()
		})
		r.Use(RoleMiddleware("admin", "superuser"))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "acesso negado")
	})

	t.Run("Empty Roles List", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("role", "admin")
			c.Next()
		})
		r.Use(RoleMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "acesso negado")
	})
}
