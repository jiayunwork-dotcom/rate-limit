package middleware

import (
	"net/http"
	"testing"
	"time"
)

func TestMiddlewareAllowsWithinLimit(t *testing.T) {
	now := time.Now()
	cfg := Config{
		Limit:  3,
		Window: time.Second,
		Clock:  func() time.Time { return now },
	}

	mw := New(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		rec := NewResponseRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i+1, rec.Code, http.StatusOK)
		}
	}

	rec := NewResponseRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestMiddlewareDifferentKeys(t *testing.T) {
	now := time.Now()
	cfg := Config{
		Limit:  1,
		Window: time.Second,
		Clock:  func() time.Time { return now },
	}

	mw := New(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	clients := []string{"10.0.0.1:1000", "10.0.0.2:2000", "10.0.0.3:3000"}
	for _, addr := range clients {
		rec := NewResponseRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req.RemoteAddr = addr
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("client %s: status = %d, want %d", addr, rec.Code, http.StatusOK)
		}
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		addr     string
		expected string
	}{
		{
			name:     "从 RemoteAddr 提取",
			addr:     "192.168.1.1:8080",
			expected: "192.168.1.1",
		},
		{
			name:     "从 X-Forwarded-For 提取",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1"},
			addr:     "192.168.1.1:8080",
			expected: "10.0.0.1",
		},
		{
			name:     "从 X-Real-IP 提取",
			headers:  map[string]string{"X-Real-IP": "172.16.0.1"},
			addr:     "192.168.1.1:8080",
			expected: "172.16.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.addr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := ExtractIP(req)
			if ip != tt.expected {
				t.Errorf("ExtractIP() = %q, want %q", ip, tt.expected)
			}
		})
	}
}
