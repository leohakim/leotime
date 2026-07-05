package httpapi

import (
	"errors"
	"net/http"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) getDashboardStats(w http.ResponseWriter, r *http.Request, user *store.User) {
	stats, err := s.store.BuildDashboardStats(r.Context(), user.ID, r.URL.Query().Get("activityMonth"))
	if err != nil {
		if errors.Is(err, store.ErrInvalidDashboardInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "load dashboard stats failed")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
