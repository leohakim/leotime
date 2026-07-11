package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type fixedWindowLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func newFixedWindowLimiter(max int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		hits:   make(map[string][]time.Time),
		max:    max,
		window: window,
	}
}

func (l *fixedWindowLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	times := l.hits[key]
	kept := times[:0]
	for _, hit := range times {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= l.max {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

func requestClientIP(r *http.Request, trustForwarded bool) string {
	if trustForwarded {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			return strings.TrimSpace(parts[0])
		}
	}
	return peerHost(r.RemoteAddr)
}

func peerHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return host
}

func (s *Server) clientIP(r *http.Request) string {
	return requestClientIP(r, s.cfg.TrustForwardedHeaders)
}

func (s *Server) rateLimitAuth(w http.ResponseWriter, r *http.Request, key string) bool {
	if s.loginLimiter.allow(key) {
		return true
	}
	writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many attempts; try again later")
	return false
}

func (s *Server) rateLimitForgotPassword(w http.ResponseWriter, r *http.Request, email string) bool {
	key := "forgot:" + s.clientIP(r) + ":" + strings.ToLower(strings.TrimSpace(email))
	if s.forgotPasswordLimiter.allow(key) {
		return true
	}
	writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many attempts; try again later")
	return false
}
