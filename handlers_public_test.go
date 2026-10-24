package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRoutes sends fake requests through the real router and checks the
// status, content type, and body of each response. httptest lets us do this
// without starting a real server or opening a network port.
func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "health check",
			method:     http.MethodGet,
			path:       "/api/health",
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "health check with wrong method",
			method:     http.MethodPost,
			path:       "/api/health",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"error":"method not allowed"}`,
		},
		{
			name:       "unknown api path",
			method:     http.MethodGet,
			path:       "/api/does-not-exist",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"not found"}`,
		},
	}

	router := newRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want JSON", got)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}
