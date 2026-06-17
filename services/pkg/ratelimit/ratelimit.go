// Package ratelimit provides Redis-backed token bucket rate limiting as Fiber middleware.
package ratelimit

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Store is the interface for rate-limit state storage (Redis).
type Store interface {
	// Take consumes one token. Returns (allowed, remaining, error).
	Take(ctx context.Context, key string, maxTokens int, window time.Duration) (bool, int, error)
}

// Config holds rate limiter configuration.
type Config struct {
	Store      Store
	Max        int           // max requests per window
	Window     time.Duration // sliding window duration
	KeyFunc    func(c *fiber.Ctx) string
	SkipFunc   func(c *fiber.Ctx) bool
}

// New creates a rate limiting Fiber middleware.
func New(cfg Config) fiber.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = defaultKeyFunc
	}
	if cfg.Max <= 0 {
		cfg.Max = 100
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}

	return func(c *fiber.Ctx) error {
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			return c.Next()
		}

		key := cfg.KeyFunc(c)
		allowed, remaining, err := cfg.Store.Take(c.Context(), key, cfg.Max, cfg.Window)
		if err != nil {
			// Fail open — log would go here in production
			return c.Next()
		}

		c.Set("X-RateLimit-Limit", itoa(cfg.Max))
		c.Set("X-RateLimit-Remaining", itoa(remaining))

		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    43001,
					"message": "rate limit exceeded, try again later",
				},
			})
		}

		return c.Next()
	}
}

func defaultKeyFunc(c *fiber.Ctx) string {
	ip := c.IP()
	path := c.Path()
	return "ratelimit:" + ip + ":" + path
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	b := make([]byte, 0, 8)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
