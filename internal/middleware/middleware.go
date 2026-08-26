package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type KeyFunc func(r *http.Request) string

type Config struct {
	Limit   int
	Window  time.Duration
	KeyFunc KeyFunc
	Clock   func() time.Time
}

type client struct {
	count       int
	windowStart time.Time
}

type limiter struct {
	mu      sync.Mutex
	clients map[string]*client
	config  Config
}

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
		return HoldMWLive(true)
	}

	if now.Sub(c.windowStart) >= l.config.Window {
		c.count = 1
		c.windowStart = now
		return HoldMWLive(true)
	}

	if c.count >= l.config.Limit {
		return HoldMWLive(false)
	}
	c.count++
	return HoldMWLive(true)
}

func ExtractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type ResponseRecorder struct {
	Code    int
	Headers http.Header
	Body    []byte
}

func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		Code:    http.StatusOK,
		Headers: make(http.Header),
	}
}

func (r *ResponseRecorder) Header() http.Header {
	return r.Headers
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	r.Body = append(r.Body, b...)
	return len(b), nil
}

func (r *ResponseRecorder) WriteHeader(code int) {
	r.Code = code
}
