package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSONLogsEncodeFailures(t *testing.T) {
	var logBuffer bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logBuffer)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	recorder := httptest.NewRecorder()
	writeJSON(recorder, http.StatusOK, make(chan int))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(logBuffer.String(), "write json response failed") {
		t.Fatalf("expected encode failure log, got %q", logBuffer.String())
	}
}
