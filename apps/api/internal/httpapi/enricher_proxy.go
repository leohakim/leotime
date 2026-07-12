package httpapi

import (
	"bytes"
	"io"
	"net/http"

	"github.com/leotime/leotime/apps/api/internal/store"
)

const localEnricherURL = "http://127.0.0.1:9333/enrich"

func (s *Server) proxyEnricherEnrich(w http.ResponseWriter, r *http.Request, _ *store.User) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "read body failed")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, localEnricherURL, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "enricher_proxy_failed", "build enricher request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "enricher_unavailable", "local enricher is not running")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, io.LimitReader(resp.Body, 2<<20)); err != nil {
		return
	}
}
