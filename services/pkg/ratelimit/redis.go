package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis with a sliding-window algorithm.
type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Take consumes one token from the bucket for the given key.
// Uses a simplified sliding-window counter:
//   - key = "ratelimit:<key>"
//   - INCR the counter, set TTL on first call
//   - If count > max, reject.
func (s *RedisStore) Take(ctx context.Context, key string, maxTokens int, window time.Duration) (bool, int, error) {
	rk := "ratelimit:" + key

	count, err := s.client.Incr(ctx, rk).Result()
	if err != nil {
		return false, 0, err
	}

	// Set expiry on first request in the window
	if count == 1 {
		s.client.Expire(ctx, rk, window)
	}

	remaining := maxTokens - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return count <= int64(maxTokens), remaining, nil
}
