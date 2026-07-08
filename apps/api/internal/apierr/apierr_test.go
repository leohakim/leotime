package apierr_test

import (
	"encoding/json"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/apierr"
)

func TestValidationErrorJSON(t *testing.T) {
	payload := apierr.Response{Error: apierr.Validation("name", "required", "name is required")}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Fields  []struct {
				Field   string `json:"field"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Error.Code != "validation_failed" {
		t.Fatalf("unexpected code: %s", decoded.Error.Code)
	}
	if len(decoded.Error.Fields) != 1 || decoded.Error.Fields[0].Field != "name" {
		t.Fatalf("unexpected fields: %+v", decoded.Error.Fields)
	}
}
