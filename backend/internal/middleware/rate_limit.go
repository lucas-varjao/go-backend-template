// backend/internal/middleware/rate_limit.go

package middleware

import (
	"net/http"
	"time"

	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips    map[string]*rate.Limiter
	mu     *sync.RWMutex
	rate   rate.Limit
	burst  int
	expiry time.Duration
}

func NewIPRateLimiter(r rate.Limit, b int, expiry time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		ips:    make(map[string]*rate.Limiter),
		mu:     &sync.RWMutex{},
		rate:   r,
		burst:  b,
		expiry: expiry,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.ips[ip] = limiter
		i.mu.Unlock()

		// Limpar IPs expirados para evitar vazamento de memória
		go func(ip string) {
			time.Sleep(i.expiry)
			i.mu.Lock()
			delete(i.ips, ip)
			i.mu.Unlock()
		}(ip)
	}

	return limiter
}

func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.GetLimiter(ip)

		if !l.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "limite de requisições excedido",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
