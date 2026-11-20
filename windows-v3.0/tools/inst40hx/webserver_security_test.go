package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyWebHandler(t *testing.T) {
	const addr = "127.0.0.1:50921"
	const origin = "http://" + addr
	tests := []struct {
		name      string
		method    string
		path      string
		host      string
		origins   []string
		fetchSite string
		want      int
		wantRoute string
	}{
		{name: "static navigation", method: http.MethodGet, path: "/", fetchSite: "none", want: http.StatusOK, wantRoute: "static"},
		{name: "same origin status", method: http.MethodGet, path: "/api/status", want: http.StatusOK, wantRoute: "status"},
		{name: "same origin events", method: http.MethodGet, path: "/api/events", origins: []string{origin}, want: http.StatusOK, wantRoute: "events"},
		{name: "same origin mutation", method: http.MethodPost, path: "/api/action", origins: []string{origin}, fetchSite: "same-origin", want: http.StatusOK, wantRoute: "action"},
		{name: "wrong mutation method", method: http.MethodGet, path: "/api/action", want: http.StatusMethodNotAllowed},
		{name: "wrong status method", method: http.MethodPost, path: "/api/status", origins: []string{origin}, want: http.StatusMethodNotAllowed},
		{name: "wrong static method", method: http.MethodPost, path: "/", origins: []string{origin}, want: http.StatusMethodNotAllowed},
		{name: "unknown API path", method: http.MethodGet, path: "/api/unknown", want: http.StatusNotFound},
		{name: "wrong listener port", method: http.MethodGet, path: "/", host: "127.0.0.1:40100", want: http.StatusForbidden},
		{name: "wrong listener address", method: http.MethodGet, path: "/api/status", host: "localhost:50921", want: http.StatusForbidden},
		{name: "missing origin", method: http.MethodPost, path: "/api/action", want: http.StatusForbidden},
		{name: "wrong origin", method: http.MethodPost, path: "/api/action", origins: []string{"http://127.0.0.1:40100"}, want: http.StatusForbidden},
		{name: "ambiguous origin", method: http.MethodPost, path: "/api/action", origins: []string{origin, origin}, want: http.StatusForbidden},
		{name: "foreign event origin", method: http.MethodGet, path: "/api/events", origins: []string{"https://example.invalid"}, want: http.StatusForbidden},
		{name: "cross site events", method: http.MethodGet, path: "/api/events", fetchSite: "cross-site", want: http.StatusForbidden},
		{name: "same site distinct origin", method: http.MethodGet, path: "/api/events", fetchSite: "same-site", want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := ""
			mark := func(route string) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {
					called = route
				}
			}
			api := http.NewServeMux()
			api.HandleFunc("GET /api/status", mark("status"))
			api.HandleFunc("GET /api/events", mark("events"))
			api.HandleFunc("POST /api/action", mark("action"))
			handler := legacyWebHandler(addr, api, mark("static"))

			request := httptest.NewRequest(tt.method, origin+tt.path, nil)
			if tt.host != "" {
				request.Host = tt.host
			}
			for _, value := range tt.origins {
				request.Header.Add("Origin", value)
			}
			if tt.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", tt.fetchSite)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.want || called != tt.wantRoute {
				t.Errorf("status %d, reached %q; want %d, reached %q", response.Code, called, tt.want, tt.wantRoute)
			}
		})
	}
}

func TestLegacyRiotOptimizationNeedsConfirmation(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/riot/optimize", nil)
	requireRiotOptimizationConfirmation(response, request)
	if response.Code != http.StatusConflict {
		t.Errorf("status %d, want %d", response.Code, http.StatusConflict)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" || response.Body.Len() == 0 {
		t.Error("expected a JSON confirmation message")
	}
}
