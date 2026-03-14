package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding window rate limiter backed by Redis.
type RateLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Key by IP (in production, use authenticated user ID)
		key := fmt.Sprintf("ratelimit:%s", r.RemoteAddr)
		ctx := r.Context()
		now := time.Now().UnixMilli()

		pipe := rl.rdb.Pipeline()

		// Remove entries outside the window
		pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-rl.window.Milliseconds()))
		// Add current request
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
		// Count requests in window
		countCmd := pipe.ZCard(ctx, key)
		// Set TTL on the key
		pipe.Expire(ctx, key, rl.window)

		if _, err := pipe.Exec(ctx); err != nil {
			// If Redis is down, allow the request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		count := countCmd.Val()
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(rl.limit)-count)))

		if count > int64(rl.limit) {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
