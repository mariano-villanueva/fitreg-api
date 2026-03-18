package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	max     int
	window  time.Duration
}

func newIPRateLimiter(max int, window time.Duration) *ipRateLimiter {
	l := &ipRateLimiter{
		entries: make(map[string][]time.Time),
		max:     max,
		window:  window,
	}
	go l.cleanup()
	return l
}

// allow returns true if the request from ip is within the rate limit.
func (l *ipRateLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.entries[ip]
	// Drop timestamps outside the sliding window.
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= l.max {
		l.entries[ip] = valid
		return false
	}
	l.entries[ip] = append(valid, now)
	return true
}

// cleanup removes stale entries every 5 minutes to prevent unbounded memory growth.
func (l *ipRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-l.window)
		l.mu.Lock()
		for ip, times := range l.entries {
			valid := times[:0]
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(l.entries, ip)
			} else {
				l.entries[ip] = valid
			}
		}
		l.mu.Unlock()
	}
}

// authLimiter allows up to 10 auth attempts per IP per minute.
var authLimiter = newIPRateLimiter(10, time.Minute)

// RateLimitAuth is middleware that enforces per-IP rate limiting on the auth endpoint.
// It returns 429 Too Many Requests when the limit is exceeded.
func RateLimitAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !authLimiter.allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"Too many requests, please try again later"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP, respecting X-Forwarded-For for reverse proxies.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can be a comma-separated list; use the first (original client).
		if i := len(xff); i > 0 {
			for j := 0; j < i; j++ {
				if xff[j] == ',' {
					return xff[:j]
				}
			}
			return xff
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
