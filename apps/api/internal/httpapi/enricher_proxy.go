package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/store"
)

const defaultLocalEnricherURL = "http://127.0.0.1:9333/enrich"

var localEnricherURL = defaultLocalEnricherURL

func (s *Server) proxyEnricherEnrich(w http.ResponseWriter, r *http.Request, user *store.User) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "read body failed")
		return
	}

	payload := map[string]any{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
			return
		}
	}
	delete(payload, "cursorApiKey")
	delete(payload, "aiEnabled")

	if record, err := s.store.AISettingsRecordByUserID(r.Context(), user.ID); err == nil && record != nil && record.Enabled {
		payload["aiEnabled"] = true
		if strings.TrimSpace(record.CursorAPIKeyEnc) != "" {
			if key, err := s.decryptSecret(record.CursorAPIKeyEnc); err == nil && strings.TrimSpace(key) != "" {
				payload["cursorApiKey"] = key
			}
		}
	}

	out, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "enricher_proxy_failed", "build enricher request failed")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, localEnricherURL, bytes.NewReader(out))
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
