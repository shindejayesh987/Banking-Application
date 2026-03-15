package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return mr, client
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	_, rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb, 5, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(dummyHandler))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i+1, rr.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	_, rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb, 3, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(dummyHandler))

	// The rate limiter uses time.Now().UnixMilli() as both score and sorted-set member.
	// Requests in the same millisecond share a member key, so we sleep 1ms between
	// requests to ensure each lands in a distinct millisecond slot.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:5678"
		handler.ServeHTTP(httptest.NewRecorder(), req)
		time.Sleep(time.Millisecond)
	}

	// 4th request should be blocked (count=4 > limit=3)
	time.Sleep(time.Millisecond)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: got %d, want 429", rr.Code)
	}
}

func TestRateLimiter_SetsHeaders(t *testing.T) {
	_, rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb, 10, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(dummyHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:9999"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	limit := rr.Header().Get("X-RateLimit-Limit")
	remaining := rr.Header().Get("X-RateLimit-Remaining")

	if limit != "10" {
		t.Errorf("X-RateLimit-Limit: got %q, want 10", limit)
	}
	limitVal, _ := strconv.Atoi(limit)
	remainingVal, _ := strconv.Atoi(remaining)
	if remainingVal != limitVal-1 {
		t.Errorf("X-RateLimit-Remaining: got %d, want %d", remainingVal, limitVal-1)
	}
}

func TestRateLimiter_ReturnsRetryAfterOnBlock(t *testing.T) {
	_, rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb, 1, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(dummyHandler))

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4:80"
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	time.Sleep(time.Millisecond) // ensure distinct ms slot

	// Second request — blocked
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "1.2.3.4:80"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set on 429 response")
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	_, rdb := newTestRedis(t)
	rl := NewRateLimiter(rdb, 1, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(dummyHandler))

	for _, ip := range []string{"1.1.1.1:80", "2.2.2.2:80", "3.3.3.3:80"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("IP %s: got %d, want 200 (first request should pass)", ip, rr.Code)
		}
	}
}

func TestRateLimiter_FailOpen(t *testing.T) {
	// Use a Redis client pointing to a dead address — should fail open (allow requests)
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:19999", DialTimeout: 50 * time.Millisecond})
	defer rdb.Close()

	rl := NewRateLimiter(rdb, 5, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(dummyHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should pass through (fail open)
	if rr.Code != http.StatusOK {
		t.Errorf("fail-open: got %d, want 200", rr.Code)
	}
}
