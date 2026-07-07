package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPasswordValidationMiddleware(t *testing.T) {
	// .test. => LnRlc3Qu

	for _, testCase := range []struct {
		input              *http.Request
		validate           func(t *testing.T, origPasswd []byte) bool
		expectedStatusCode int
	}{
		{
			httptest.NewRequest("GET", "/", nil),
			func(t *testing.T, origPasswd []byte) bool {
				t.Errorf("function should not be called")
				return false
			},
			401,
		},
		{
			httptest.NewRequest("GET", "/?passwd=LnRlc3Qu", nil),
			func(t *testing.T, origPasswd []byte) bool {
				if !bytes.Equal(origPasswd, []byte(".test.")) {
					t.Errorf("password missmatch: %s != \".test.\"", origPasswd)
					return false
				}
				return true
			},
			200,
		},
		{
			httptest.NewRequest("GET", "/?passwd=LnRlc3Qu", nil),
			func(t *testing.T, origPasswd []byte) bool {
				return false
			},
			401,
		},
		{
			httptest.NewRequest("GET", "/?passwd=2mlFWmE8HeqGclB9vCu7k8uoSEKfXXxTSpGnnEBvBPs", nil),
			func(t *testing.T, origPasswd []byte) bool {
				b, err := base64.RawURLEncoding.DecodeString("2mlFWmE8HeqGclB9vCu7k8uoSEKfXXxTSpGnnEBvBPs")
				if err != nil {
					t.Fatalf("cannot decode hard-coded string: %v", err)
				}

				if !bytes.Equal(origPasswd, b) {
					t.Errorf("got %v instead of %v", origPasswd, b)
				}
				return true
			},
			200,
		},
	} {
		route := chi.NewRouter()
		route.Use(PasswordValidationMiddleware(func(origPasswd []byte) bool {
			return testCase.validate(t, origPasswd)
		}))
		route.Get("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Ok")
		})

		w := httptest.NewRecorder()
		route.ServeHTTP(w, testCase.input)
		resp := w.Result()
		if resp.StatusCode != testCase.expectedStatusCode {
			t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatusCode)
		}
	}
}

func TestUserValidationMiddleware(t *testing.T) {

	for _, testCase := range []struct {
		input              *http.Request
		expectedStatusCode int
	}{
		{httptest.NewRequest("GET", "/", nil), 401},
		{httptest.NewRequest("GET", "/?user=", nil), 401},
		{httptest.NewRequest("GET", "/?user=foobar", nil), 401},
		{httptest.NewRequest("GET", "/?user=baz", nil), 200},
	} {
		route := chi.NewRouter()
		route.Use(UserValidationMiddleware("baz"))
		route.Get("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Ok")
		})

		w := httptest.NewRecorder()
		route.ServeHTTP(w, testCase.input)
		resp := w.Result()
		if resp.StatusCode != testCase.expectedStatusCode {
			t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatusCode)
		}
	}
}

func TestIPValidationMiddleware(t *testing.T) {
	checkContext := func(t *testing.T, r *http.Request, ctxKey ctxIPKey, expectedValue *netip.Addr) {
		t.Helper()
		got, ok := r.Context().Value(ctxKey).(*netip.Addr)
		if expectedValue != nil && !ok {
			t.Errorf("expected value but got nothing")
		}
		if expectedValue == nil && ok {
			t.Errorf("expected nothing but got %v", got)
		}

		if expectedValue != nil && *expectedValue != *got {
			t.Errorf("expected %v but got: %v", expectedValue, got)
		}
	}

	ipv4 := netip.MustParseAddr("192.168.1.1")
	ipv6 := netip.MustParseAddr("2001:db8::1")

	for _, testCase := range []struct {
		name               string
		input              *http.Request
		expectedStatusCode int
		expectedIPv4       *netip.Addr
		expectedIPv6       *netip.Addr
	}{
		{"empty", httptest.NewRequest("GET", "/", nil), 200, nil, nil},
		{"v4 only", httptest.NewRequest("GET", "/?ipaddr=192.168.1.1", nil), 200, &ipv4, nil},
		{"v6 only", httptest.NewRequest("GET", "/?ip6addr=2001:db8::1", nil), 200, nil, &ipv6},
		{"both", httptest.NewRequest("GET", "/?ipaddr=192.168.1.1&ip6addr=2001:db8::1", nil), 200, &ipv4, &ipv6},

		{"invalid v4", httptest.NewRequest("GET", "/?ipaddr=a.168.1.1", nil), 400, nil, nil},
		{"invalid v6", httptest.NewRequest("GET", "/?ip6addr=2001:db8::x", nil), 400, nil, nil},
		{"invalid v4 with valid v6", httptest.NewRequest("GET", "/?ipaddr=a.168.1.1&ip6addr=2001:db8::1", nil), 400, nil, nil},
		{"valid v4 with invalid v6", httptest.NewRequest("GET", "/?ipaddr=192.168.1.1&ip6addr=2001:db8::x", nil), 400, nil, nil},

		// Family-mismatch: IPv6 value in the v4 slot and vice versa.
		{"v6 in ipaddr slot", httptest.NewRequest("GET", "/?ipaddr=2001:db8::1", nil), 400, nil, nil},
		{"v4 in ip6addr slot", httptest.NewRequest("GET", "/?ip6addr=192.168.1.1", nil), 400, nil, nil},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			nextCalled := false
			route := chi.NewRouter()
			route.Use(IPValidationMiddleware)
			route.Get("/", func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				checkContext(t, r, ctxIPv4Key, testCase.expectedIPv4)
				checkContext(t, r, ctxIPv6Key, testCase.expectedIPv6)
			})

			w := httptest.NewRecorder()
			route.ServeHTTP(w, testCase.input)
			resp := w.Result()
			if resp.StatusCode != testCase.expectedStatusCode {
				t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatusCode)
			}
			if resp.StatusCode >= 400 && nextCalled {
				t.Errorf("next handler was invoked after an error response")
			}
		})
	}
}

type mockZonefileWriter struct {
	s          subdomain
	setCalled  bool
	writeCalled bool
	checkWrite func(m *mockZonefileWriter, wr io.Writer) error
}

func (m *mockZonefileWriter) Set(s subdomain) {
	m.s = s
	m.setCalled = true

}
func (m *mockZonefileWriter) Write(wr io.Writer) error {
	m.writeCalled = true
	return m.checkWrite(m, wr)
}

func TestZonefileWriteHandler(t *testing.T) {
	ctx := context.Background()
	ipv4 := netip.MustParseAddr("192.168.1.1")
	ipv6 := netip.MustParseAddr("2001:db8::1")

	stale := []byte("STALE-CONTENT-THAT-MUST-NOT-SURVIVE\n")
	tmpMissing := filepath.Join(t.TempDir(), "does", "not", "exist", "zone.txt")

	for _, testCase := range []struct {
		name               string
		filePath           func(t *testing.T) string
		ctx                context.Context
		writeError         error
		wantStatus         int
		wantBody           string
		wantTruncate       bool
		wantStaleSurvives  bool
		wantWriterUntouched bool
	}{
		{
			name:         "v4 only",
			filePath:     func(t *testing.T) string { return freshTempWithStale(t, stale) },
			ctx:          context.WithValue(ctx, ctxIPv4Key, &ipv4),
			wantStatus:   http.StatusOK,
			wantBody:     "Ok",
			wantTruncate: true,
		},
		{
			name:         "v6 only",
			filePath:     func(t *testing.T) string { return freshTempWithStale(t, stale) },
			ctx:          context.WithValue(ctx, ctxIPv6Key, &ipv6),
			wantStatus:   http.StatusOK,
			wantBody:     "Ok",
			wantTruncate: true,
		},
		{
			name:         "v4 and v6",
			filePath:     func(t *testing.T) string { return freshTempWithStale(t, stale) },
			ctx:          context.WithValue(context.WithValue(ctx, ctxIPv6Key, &ipv6), ctxIPv4Key, &ipv4),
			wantStatus:   http.StatusOK,
			wantBody:     "Ok",
			wantTruncate: true,
		},
		{
			// Both addresses missing: we acknowledge the request with Ok
			// but do not touch the zonefile or the writer, so an existing
			// zone survives untouched.
			name:               "v4 and v6 both nil",
			filePath:           func(t *testing.T) string { return freshTempWithStale(t, stale) },
			ctx:                context.Background(),
			wantStatus:         http.StatusOK,
			wantBody:           "Ok",
			wantStaleSurvives:  true,
			wantWriterUntouched: true,
		},
		{
			// Errors are logged internally; the client always sees Ok.
			name:         "write error",
			filePath:     func(t *testing.T) string { return freshTempWithStale(t, stale) },
			writeError:   fmt.Errorf("disk on fire"),
			wantStatus:   http.StatusOK,
			wantBody:     "Ok",
			wantTruncate: false,
		},
		{
			name:       "open error",
			filePath:   func(t *testing.T) string { return tmpMissing },
			wantStatus: http.StatusOK,
			wantBody:   "Ok",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			path := testCase.filePath(t)
			writer := &mockZonefileWriter{
				checkWrite: func(m *mockZonefileWriter, wr io.Writer) error {
					return testCase.writeError
				},
			}

			routeCtx := testCase.ctx
			if routeCtx == nil {
				routeCtx = context.Background()
			}

			route := chi.NewRouter()
			route.Get("/", ZonefileWriteHandler(path, "example", writer))

			w := httptest.NewRecorder()
			route.ServeHTTP(w, httptest.NewRequest("GET", "/", nil).WithContext(routeCtx))
			resp := w.Result()
			if resp.StatusCode != testCase.wantStatus {
				t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.wantStatus)
			}
			body, _ := io.ReadAll(resp.Body)
			if got := string(bytes.TrimSpace(body)); got != testCase.wantBody {
				t.Errorf("response body is %q instead of %q", got, testCase.wantBody)
			}

			if testCase.wantTruncate {
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(got, stale) {
					t.Errorf("stale content was not truncated: file contents: %q", got)
				}
			}

			if testCase.wantStaleSurvives {
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(got, stale) {
					t.Errorf("stale content was lost; zonefile should have been left untouched: %q", got)
				}
			}

			if testCase.wantWriterUntouched {
				if writer.setCalled {
					t.Errorf("writer.Set was called; the empty-update guard should short-circuit before the writer runs")
				}
				if writer.writeCalled {
					t.Errorf("writer.Write was called; the empty-update guard should short-circuit before the writer runs")
				}
			}
		})
	}
}

// freshTempWithStale returns a path to a temp file pre-filled with stale,
// so O_TRUNC is observable.
func freshTempWithStale(t *testing.T, stale []byte) string {
	t.Helper()
	file, err := os.CreateTemp("", "zone-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(file.Name()) })

	if _, err := file.Write(stale); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}

func TestHttpRouter(t *testing.T) {
	route := chi.NewRouter()
	route.Mount("/", updaterHandler(updaterHandlerConfig{}))

	// Without valid credentials the user-validation middleware returns 401,
	// proving the router is fully wired.
	w := httptest.NewRecorder()
	route.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("status code is %v instead of %v", w.Result().StatusCode, http.StatusUnauthorized)
	}
}
