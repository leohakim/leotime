package httpapi

import (
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
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")) == token
	}
	return strings.TrimSpace(r.URL.Query().Get("token")) == token
}
