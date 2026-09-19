package middleware

import (
	"net/http"
	"sync"
	"time"

	"gin-starter-pack/config"
	"gin-starter-pack/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientLimiter
	rps     rate.Limit
	burst   int
	enabled bool
}

// NewIPRateLimiter creates a new rate limiter per IP
func NewIPRateLimiter(cfg config.RateLimitConfig) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		rps:     rate.Limit(cfg.RequestsPerSecond),
		burst:   cfg.Burst,
		enabled: cfg.Enabled,
	}

	// Periodically clean up stale client entries (older than 3 minutes)
	if cfg.Enabled {
		go limiter.cleanupStaleClients(3 * time.Minute)
	}

	return limiter
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	c, exists := l.clients[ip]
	if !exists {
		limiter := rate.NewLimiter(l.rps, l.burst)
		l.clients[ip] = &clientLimiter{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (l *IPRateLimiter) cleanupStaleClients(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		l.mu.Lock()
		for ip, client := range l.clients {
			if time.Since(client.lastSeen) > interval {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

// Handler returns the Gin middleware HandlerFunc
func (l *IPRateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.enabled {
			c.Next()
			return
		}

		ip := c.ClientIP()
		limiter := l.getLimiter(ip)

		if !limiter.Allow() {
			response.Error(c, http.StatusTooManyRequests, "Too many requests. Please try again later.", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
