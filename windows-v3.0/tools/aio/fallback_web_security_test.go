package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFallbackMutationGuard(t *testing.T) {
	called := false
	handler := fallbackMutationGuard(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	for _, tt := range []struct {
		name   string
		method string
		path   string
		origin string
		want   int
	}{
		{name: "cross-origin denied", method: "POST", path: "/api/uninstall", origin: "http://attacker.example", want: http.StatusForbidden},
		{name: "same-origin accepted", method: "POST", path: "/api/uninstall", origin: "http://127.0.0.1:40100", want: http.StatusOK},
		{name: "cross-origin SSE denied", method: "GET", path: "/api/events/stream", origin: "http://attacker.example", want: http.StatusForbidden},
		{name: "same-origin SSE accepted", method: "GET", path: "/api/events/stream", origin: "http://127.0.0.1:40100", want: http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			called = false
			r := httptest.NewRequest(tt.method, "http://127.0.0.1:40100"+tt.path, nil)
			r.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
			if called != (tt.want == http.StatusOK) {
				t.Fatalf("handler called = %t, want %t", called, tt.want == http.StatusOK)
			}
		})
	}
}

func TestRequireFallbackConfirmation(t *testing.T) {
	for _, tt := range []struct {
		name string
		ok   bool
		want int
	}{
		{name: "accept", ok: true, want: http.StatusOK},
		{name: "deny", ok: false, want: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			if got := requireFallbackConfirmation(w, func() bool { return tt.ok }); got != tt.ok {
				t.Fatalf("confirmed = %t, want %t", got, tt.ok)
			}
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestValidFallbackMutation(t *testing.T) {
	const allowedOrigin = "http://127.0.0.1:40100"
	tests := []struct {
		name   string
		method string
		origin string
		host   string
		want   bool
	}{
		{name: "same-origin request", method: "POST", origin: allowedOrigin, host: "127.0.0.1:40100", want: true},
		{name: "reject cross-origin", method: "POST", origin: "http://attacker.example", host: "127.0.0.1:40100"},
		{name: "reject DNS rebinding host", method: "POST", origin: "http://attacker.example", host: "attacker.example"},
		{name: "reject missing origin", method: "POST", host: "127.0.0.1:40100"},
		{name: "reject wrong method", method: "GET", origin: allowedOrigin, host: "127.0.0.1:40100"},
		{name: "reject unsupported origin scheme", method: "POST", origin: "https://127.0.0.1:40100", host: "127.0.0.1:40100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://"+tt.host+"/api/uninstall", nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if got := validFallbackMutation(r); got != tt.want {
				t.Fatalf("validFallbackMutation() = %t, want %t", got, tt.want)
			}
		})
	}
}
