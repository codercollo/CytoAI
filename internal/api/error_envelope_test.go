package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteErrorEmitsEmptyDetailsObject ensures every simple error response
// carries the documented envelope with `details` as an object ({}), never null.
func TestWriteErrorEmitsEmptyDetailsObject(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusBadRequest, "invalid_request", "bad request", nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body struct {
		Error   string         `json:"error"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "invalid_request" {
		t.Errorf("error = %q, want invalid_request", body.Error)
	}
	if body.Message != "bad request" {
		t.Errorf("message = %q, want bad request", body.Message)
	}
	if body.Details == nil {
		t.Errorf("details = nil, want empty object {}; body=%s", rr.Body.String())
	}
}
