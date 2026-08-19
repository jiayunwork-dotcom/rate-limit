// Package middleware provides HTTP middleware for rate limiting.
//
// It wraps http.Handler with configurable rate limiting that can
// identify clients by IP address or custom key functions.
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// KeyFunc extracts a rate limiting key from an HTTP request.
type KeyFunc func(r *http.Request) string

// Config holds the configuration for the rate limiting middleware.
type Config struct {
	// Limit is the maximum number of requests per window.
	Limit int
	// Window is the duration of the rate limiting window.
	Window time.Duration
	// KeyFunc extracts the rate limiting key from requests.
	// If nil, ExtractIP is used by default.
	KeyFunc KeyFunc
	// Clock provides the current time (for testing).
	Clock func() time.Time
}

// client tracks rate limiting state for a single client.
type client struct {
	count       int
	windowStart time.Time
}

// limiter holds the state for all clients.
type limiter struct {
	mu      sync.Mutex
	clients map[string]*client
	config  Config
}

// New creates a new rate limiting middleware with the given configuration.
func New(cfg Config) func(http.Handler) http.Handler {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = ExtractIP
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}

	l := &limiter{
		clients: make(map[string]*client),
		config:  cfg,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := l.config.KeyFunc(r)
			if !l.allow(key) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if a request from the given key is allowed.
func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.config.Clock()
	c, ok := l.clients[key]
	if !ok {
		l.clients[key] = &client{
			count:       1,
			windowStart: now,
		}
		return true
	}

	// 检查窗口是否过期
	if now.Sub(c.windowStart) >= l.config.Window {
		c.count = 1
		c.windowStart = now
		return true
	}

	if c.count >= l.config.Limit {
		return false
	}
	c.count++
	return true
}

// ExtractIP extracts the client IP address from an HTTP request.
// It checks X-Forwarded-For and X-Real-IP headers before falling
// back to RemoteAddr.
func ExtractIP(r *http.Request) string {
	// 检查 X-Forwarded-For 头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// 检查 X-Real-IP 头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ResponseRecorder is a test helper that records HTTP response details.
type ResponseRecorder struct {
	Code    int
	Headers http.Header
	Body    []byte
}

// NewResponseRecorder creates a new ResponseRecorder.
func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Code:    http.StatusOK,
		Headers: make(http.Header),
	}
}

// Header returns the response headers.
func (r *ResponseRecorder) Header() http.Header {
	return r.Headers
}

// Write writes the response body.
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	r.Body = append(r.Body, b...)
	return len(b), nil
}

// WriteHeader records the status code.
func (r *ResponseRecorder) WriteHeader(code int) {
	r.Code = code
}
