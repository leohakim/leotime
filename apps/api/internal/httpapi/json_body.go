package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const defaultMaxJSONBodyBytes = 1 << 20

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dest any) bool {
	return decodeJSONBodyLimit(w, r, defaultMaxJSONBodyBytes, dest)
}

func decodeJSONBodyLimit(w http.ResponseWriter, r *http.Request, limit int64, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		writeJSONBodyDecodeError(w, err)
		return false
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == io.EOF {
		return true
	} else if err != nil {
		writeJSONBodyDecodeError(w, err)
		return false
	}

	writeError(w, http.StatusBadRequest, "invalid_json", "trailing json values are not allowed")
	return false
}

func writeJSONBodyDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body too large")
		return
	}
	if errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "empty json body")
		return
	}
	if strings.Contains(err.Error(), "unknown field") {
		writeError(w, http.StatusBadRequest, "invalid_json", "unknown json field")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
}
