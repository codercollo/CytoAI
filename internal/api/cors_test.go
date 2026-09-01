package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func corsTestServer(t *testing.T, origins []string, authOK bool) *Server {
	t.Helper()
	auth := mockAuth{authenticate: func(ctx context.Context, key string) (string, error) {
		if !authOK {
			return "", errors.New("invalid key")
		}
		return "partner-1", nil
	}}
	return NewServer(Config{
		Store:              &mockStore{},
		Scorer:             mockScorer{},
		Auth:               auth,
		CORSAllowedOrigins: origins,
	})
}

func TestCORSPreflight(t *testing.T) {
	srv := corsTestServer(t, []string{"http://localhost:3000"}, true)

	req := httptest.NewRequest(http.MethodOptions, "/v1/telematics", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://localhost:3000", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %q, want GET, POST, OPTIONS", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Errorf("Access-Control-Allow-Headers = %q, want Authorization, Content-Type", got)
	}
}

func TestCORSPreflight_DisallowedOrigin(t *testing.T) {
	srv := corsTestServer(t, []string{"http://localhost:3000"}, true)

	req := httptest.NewRequest(http.MethodOptions, "/v1/telematics", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty for disallowed origin", got)
	}
}

func TestCORSHeadersOnAuthenticatedRequest(t *testing.T) {
	srv := corsTestServer(t, []string{"http://localhost:3000"}, true)

	req := httptest.NewRequest(http.MethodGet, "/v1/portfolio", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Authorization", "Bearer good")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://localhost:3000", got)
	}
}

func TestAuthStillEnforcedWithCORS(t *testing.T) {
	srv := corsTestServer(t, []string{"http://localhost:3000"}, false)

	req := httptest.NewRequest(http.MethodGet, "/v1/portfolio", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Authorization", "Bearer bad")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rr.Code, rr.Body.String())
	}
}
