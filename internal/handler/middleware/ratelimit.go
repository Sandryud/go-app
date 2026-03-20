package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"workout-app/internal/handler/response"
)

// PublicShareRateLimit ограничивает число запросов с одного IP (токен-бакет).
// При rps <= 0 middleware ничего не ограничивает.
func PublicShareRateLimit(rps float64, burst int) gin.HandlerFunc {
	if rps <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	if burst < 1 {
		burst = 1
	}
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		lim, ok := limiters[ip]
		if !ok {
			lim = rate.NewLimiter(rate.Limit(rps), burst)
			limiters[ip] = lim
		}
		mu.Unlock()
		if !lim.Allow() {
			response.Error(c, http.StatusTooManyRequests, "rate_limit_exceeded", "Слишком много запросов, попробуйте позже", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
