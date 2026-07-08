package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/apierr"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func writeAPIError(w http.ResponseWriter, status int, body apierr.Error) {
	writeJSON(w, status, apierr.Response{Error: body})
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeAPIError(w, status, apierr.Simple(code, message))
}

func writeValidationStoreError(w http.ResponseWriter, err error) {
	var validation *store.ValidationError
	if errors.As(err, &validation) {
		writeAPIError(w, http.StatusBadRequest, apierr.Validation(validation.Field, validation.Code, validation.Msg))
		return
	}

	writeAPIError(w, http.StatusBadRequest, apierr.Error{
		Code:    "validation_failed",
		Message: validationDetail(err),
	})
}

func validationDetail(err error) string {
	message := err.Error()
	if idx := strings.Index(message, ": "); idx >= 0 {
		return message[idx+2:]
	}
	return message
}
