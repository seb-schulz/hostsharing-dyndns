package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestRejectBotsMiddleware verifies the cheap pre-filter that keeps bot and
// scanner traffic from triggering argon2id work and zonefile rewrites.
//
// A request reaches the next handler only if ALL of the following hold:
//   - method is GET
//   - path is exactly "/"
//   - the "user" query parameter is non-empty
//
// The User-Agent header is intentionally NOT checked here: any client
// (curl, browser debugger, etc.) that satisfies the three cheap checks is
// forwarded to the authoritative auth gate in updater.go.
func TestRejectBotsMiddleware(t *testing.T) {
	for _, testCase := range []struct {
		name             string
		method           string
		path             string
		query            string
		expectedStatus   int
		expectNextCalled bool
		expectEmptyBody  bool
	}{
		{
			name:             "valid request reaches next handler",
			method:           "GET",
			path:             "/",
			query:            "?user=foo&passwd=bar",
			expectedStatus:   http.StatusOK,
			expectNextCalled: true,
		},
		{
			name:            "non-GET rejected",
			method:          "POST",
			path:            "/",
			query:           "",
			expectedStatus:  http.StatusForbidden,
			expectEmptyBody: true,
		},
		{
			name:            "scanner path rejected",
			method:          "GET",
			path:            "/.git/config",
			query:           "?user=foo&passwd=bar",
			expectedStatus:  http.StatusForbidden,
			expectEmptyBody: true,
		},
		{
			name:            "missing user query rejected",
			method:          "GET",
			path:            "/",
			query:           "?passwd=bar",
			expectedStatus:  http.StatusForbidden,
			expectEmptyBody: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			nextCalled := false
			route := chi.NewRouter()
			route.Use(RejectBotsMiddleware)
			route.Get("/", func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			route.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("405 handler should not be reached; RejectBots must short-circuit first")
			})

			req := httptest.NewRequest(testCase.method, testCase.path+testCase.query, nil)
			w := httptest.NewRecorder()
			route.ServeHTTP(w, req)
			resp := w.Result()

			if resp.StatusCode != testCase.expectedStatus {
				t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatus)
			}
			if nextCalled != testCase.expectNextCalled {
				t.Errorf("next handler called=%v, expected=%v", nextCalled, testCase.expectNextCalled)
			}

			if testCase.expectEmptyBody {
				body, _ := io.ReadAll(resp.Body)
				if len(body) != 0 {
					t.Errorf("expected empty body, got %d bytes: %q", len(body), body)
				}
				if cl := resp.Header.Get("Content-Length"); cl != "" && cl != "0" {
					t.Errorf("expected no/zero Content-Length, got %q", cl)
				}
			}
		})
	}
}

// TestRejectBotsMiddleware_DoesNotPanicOnFlood pins the wire shape: even
// under flood, rejections stay 403 with an empty body and never leak the
// password-error text.
func TestRejectBotsMiddleware_DoesNotPanicOnFlood(t *testing.T) {
	route := chi.NewRouter()
	route.Use(RejectBotsMiddleware)
	route.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/.env?user=foo&passwd=bar", nil)
		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusForbidden {
			t.Fatalf("iteration %d: expected 403, got %d", i, w.Result().StatusCode)
		}
		if strings.Contains(w.Body.String(), "user or password wrong") {
			t.Fatalf("iteration %d: rejected body must not leak password-error text", i)
		}
	}
}
