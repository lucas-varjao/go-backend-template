// backend/internal/middleware/rate_limit_test.go

package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestIPRateLimiter(t *testing.T) {
	t.Run("GetLimiter Creates New Limiter", func(t *testing.T) {
		// Setup
		limiter := NewIPRateLimiter(1, 5, time.Minute)
		ip := "192.168.1.1"

		// Test getting a new limiter
		result := limiter.GetLimiter(ip)

		// Assert
		assert.NotNil(t, result)

		// Check internal state
		limiter.mu.RLock()
		storedLimiter, exists := limiter.ips[ip]
		limiter.mu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, result, storedLimiter)
	})

	t.Run("GetLimiter Returns Existing Limiter", func(t *testing.T) {
		// Setup
		limiter := NewIPRateLimiter(1, 5, time.Minute)
		ip := "192.168.1.2"

		// Get limiter for the first time
		first := limiter.GetLimiter(ip)
		// Get limiter for the second time
		second := limiter.GetLimiter(ip)

		// Assert we get the same instance
		assert.Equal(t, first, second)
	})

	t.Run("Limiter Expires After Duration", func(t *testing.T) {
		// Setup with short expiry time for testing
		expiry := 100 * time.Millisecond
		limiter := NewIPRateLimiter(1, 5, expiry)
		ip := "192.168.1.3"

		// Get a limiter
		limiter.GetLimiter(ip)

		// Verify the limiter exists
		limiter.mu.RLock()
		_, exists := limiter.ips[ip]
		limiter.mu.RUnlock()
		assert.True(t, exists)

		// Wait for expiration (slightly longer than expiry)
		time.Sleep(expiry + 50*time.Millisecond)

		// Verify the limiter is gone
		limiter.mu.RLock()
		_, exists = limiter.ips[ip]
		limiter.mu.RUnlock()
		assert.False(t, exists)
	})

	t.Run("Different IPs Get Different Limiters", func(t *testing.T) {
		// Setup
		ipLimiter := NewIPRateLimiter(1, 1, time.Minute) // Rate of 1 request per second, burst of 1
		ip1 := "192.168.1.4"
		ip2 := "192.168.1.5"

		// Get limiters for different IPs
		limiter1 := ipLimiter.GetLimiter(ip1)
		limiter2 := ipLimiter.GetLimiter(ip2)

		// Use up the first limiter's quota
		assert.True(t, limiter1.Allow(), "First request from IP1 should be allowed")
		assert.False(t, limiter1.Allow(), "Second request from IP1 should be denied")

		// The second limiter should still allow requests
		assert.True(t, limiter2.Allow(), "First request from IP2 should be allowed regardless of IP1's limit")

		// Verify the limiters are stored separately
		ipLimiter.mu.RLock()
		stored1 := ipLimiter.ips[ip1]
		stored2 := ipLimiter.ips[ip2]
		ipLimiter.mu.RUnlock()

		assert.Same(t, limiter1, stored1, "Limiter1 should be the same instance as stored")
		assert.Same(t, limiter2, stored2, "Limiter2 should be the same instance as stored")
		assert.NotSame(t, stored1, stored2, "Stored limiters should be different instances")
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Allow Request Under Limit", func(t *testing.T) {
		// Setup a rate limiter with high limit
		ipLimiter := NewIPRateLimiter(10, 10, time.Minute)

		// Create test router with middleware and a flag to track if handler was called
		handlerCalled := false

		r := gin.New()
		r.Use(RateLimitMiddleware(ipLimiter))
		r.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		})

		// Make request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.10") // Set IP
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Block Request Over Limit", func(t *testing.T) {
		// Setup a rate limiter with very low limit
		ipLimiter := NewIPRateLimiter(0.1, 1, time.Minute)

		// Create test router with middleware
		handlerCalled := false

		r := gin.New()
		r.Use(RateLimitMiddleware(ipLimiter))
		r.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		})

		// Manually exhaust the limiter for the IP
		ip := "192.168.1.20"
		limiter := ipLimiter.GetLimiter(ip)
		limiter.Allow() // Use up the single token

		// Make request
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", ip)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "limite de requisições excedido")
	})

	t.Run("Multiple Requests From Same IP", func(t *testing.T) {
		// Setup a rate limiter with 2 requests per minute and burst of 2
		ipLimiter := NewIPRateLimiter(rate.Every(30*time.Second), 2, time.Minute)

		// Create test router with middleware
		r := gin.New()
		r.Use(RateLimitMiddleware(ipLimiter))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		ip := "192.168.1.30"

		// Function to make a test request
		makeRequest := func() (int, string) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", ip)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			return w.Code, w.Body.String()
		}

		// First request should be allowed
		code1, _ := makeRequest()
		assert.Equal(t, http.StatusOK, code1)

		// Second request should be allowed (within burst limit)
		code2, _ := makeRequest()
		assert.Equal(t, http.StatusOK, code2)

		// Third request should be blocked (exceeds burst)
		code3, body3 := makeRequest()
		assert.Equal(t, http.StatusTooManyRequests, code3)
		assert.Contains(t, body3, "limite de requisições excedido")
	})

	t.Run("Concurrent Requests Handling", func(t *testing.T) {
		// Setup a rate limiter with moderate limits
		ipLimiter := NewIPRateLimiter(10, 5, time.Minute)

		// Create test router with middleware
		r := gin.New()
		r.Use(RateLimitMiddleware(ipLimiter))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		ip := "192.168.1.40"

		// Use WaitGroup to ensure all goroutines complete
		var wg sync.WaitGroup
		requestCount := 10
		results := make([]bool, requestCount)

		for i := 0; i < requestCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-For", ip)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				results[index] = w.Code == http.StatusOK
			}(i)
		}

		wg.Wait()

		// Count allowed requests
		allowed := 0
		for _, passed := range results {
			if passed {
				allowed++
			}
		}

		// We expect the burst capacity (5) to be allowed
		assert.LessOrEqual(t, allowed, 5)
	})

	t.Run("Different IPs Have Separate Rate Limits", func(t *testing.T) {
		// Setup a rate limiter with 1 burst per IP
		ipLimiter := NewIPRateLimiter(0.1, 1, time.Minute)

		// Create test router with middleware
		r := gin.New()
		r.Use(RateLimitMiddleware(ipLimiter))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		ip1 := "192.168.1.50"
		ip2 := "192.168.1.51"

		// Function to make a test request
		makeRequest := func(ip string) (int, string) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", ip)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			return w.Code, w.Body.String()
		}

		// First request from IP1 should be allowed
		code1, _ := makeRequest(ip1)
		assert.Equal(t, http.StatusOK, code1)

		// Second request from IP1 should be blocked (exceeds the limit)
		code2, body2 := makeRequest(ip1)
		assert.Equal(t, http.StatusTooManyRequests, code2)
		assert.Contains(t, body2, "limite de requisições excedido")

		// First request from IP2 should still be allowed (different IP)
		code3, _ := makeRequest(ip2)
		assert.Equal(t, http.StatusOK, code3)
	})
}
