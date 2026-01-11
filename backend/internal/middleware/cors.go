// backend/internal/middleware/cors.go

package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CorsMiddleware configures CORS for the API
func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		// Explicitly allow common dev origins (localhost and 127.0.0.1 with typical Vite/Bun ports)
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:4173", "http://127.0.0.1:5173", "http://127.0.0.1:4173"},
		// Additionally accept origins programmatically if they match localhost or 127.0.0.1 on any port
		AllowOriginFunc: func(origin string) bool {
			// simple check to allow http://localhost:<port> and http://127.0.0.1:<port>
			if origin == "" {
				return false
			}
			if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
