package httpapi

import (
	"crypto/subtle"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/config"
)

func safeStaticFilePath(staticDir, requestPath string) (string, bool) {
	if staticDir == "" {
		return "", false
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(requestPath, "/"))
	if cleanPath == "." {
		cleanPath = "index.html"
	}

	root, err := filepath.Abs(staticDir)
	if err != nil {
		return "", false
	}
	fullPath, err := filepath.Abs(filepath.Join(root, cleanPath))
	if err != nil {
		return "", false
	}

	rel, err := filepath.Rel(root, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}

	return fullPath, true
}

func metricsAuthorized(cfg config.Config, r *http.Request) bool {
	token := strings.TrimSpace(cfg.MetricsToken)
	if token == "" {
		return !cfg.Production
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	return secureTokenMatch(got, token)
}

func secureTokenMatch(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
