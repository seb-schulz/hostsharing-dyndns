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
	checkContext := func(r *http.Request, ctxKey ctxIPKey, expectedValue *netip.Addr) {
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
		input              *http.Request
		expectedStatusCode int
		expectedIPv4       *netip.Addr
		expectedIPv6       *netip.Addr
	}{
		{httptest.NewRequest("GET", "/", nil), 200, nil, nil},
		{httptest.NewRequest("GET", "/?ipaddr=192.168.1.1", nil), 200, &ipv4, nil},
		{httptest.NewRequest("GET", "/?ip6addr=2001:db8::1", nil), 200, nil, &ipv6},
		{httptest.NewRequest("GET", "/?ipaddr=192.168.1.1&ip6addr=2001:db8::1", nil), 200, &ipv4, &ipv6},

		{httptest.NewRequest("GET", "/?ipaddr=a.168.1.1", nil), 400, nil, nil},
		{httptest.NewRequest("GET", "/?ip6addr=2001:db8::x", nil), 400, nil, nil},
		{httptest.NewRequest("GET", "/?ipaddr=a.168.1.1&ip6addr=2001:db8::1", nil), 400, nil, nil},
		{httptest.NewRequest("GET", "/?ipaddr=192.168.1.1&ip6addr=2001:db8::x", nil), 400, nil, nil},
	} {
		route := chi.NewRouter()
		route.Use(IPValidationMiddleware)
		route.Get("/", func(w http.ResponseWriter, r *http.Request) {
			checkContext(r, ctxIPv4Key, testCase.expectedIPv4)
			checkContext(r, ctxIPv6Key, testCase.expectedIPv6)
		})

		w := httptest.NewRecorder()
		route.ServeHTTP(w, testCase.input)
		resp := w.Result()
		if resp.StatusCode != testCase.expectedStatusCode {
			t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatusCode)
		}
	}
}

type mockZonefileWriter struct {
	s          subdomain
	checkWrite func(m *mockZonefileWriter, wr io.Writer) error
}

func (m *mockZonefileWriter) Set(s subdomain) {
	m.s = s

}
func (m *mockZonefileWriter) Write(wr io.Writer) error {
	return m.checkWrite(m, wr)
}

func TestZonefileWriteHandler(t *testing.T) {
	ZonefileWriteHandler("fake-file.txt", "example", newZonefile())

	ctx := context.Background()
	ipv4 := netip.MustParseAddr("192.168.1.1")
	ipv6 := netip.MustParseAddr("2001:db8::1")

	for _, testCase := range []struct {
		ctx        context.Context
		checkWrite func(m *mockZonefileWriter, wr io.Writer) error
	}{
		{
			context.WithValue(ctx, ctxIPv4Key, &ipv4),
			func(m *mockZonefileWriter, wr io.Writer) error {
				if m.s.IPv4 != &ipv4 {
					t.Errorf("got %v instead of %v", m.s.IPv4, &ipv4)
				}

				if m.s.IPv6 != nil {
					t.Errorf("value was set to %v instead of nil", m.s.IPv6)
				}
				return nil
			},
		},
		{
			context.WithValue(ctx, ctxIPv6Key, &ipv6),
			func(m *mockZonefileWriter, wr io.Writer) error {
				if m.s.IPv6 != &ipv6 {
					t.Errorf("got %v instead of %v", m.s.IPv6, &ipv6)
				}

				if m.s.IPv4 != nil {
					t.Errorf("value was set to %v instead of nil", m.s.IPv4)
				}
				return nil
			},
		},
		{
			context.WithValue(context.WithValue(ctx, ctxIPv6Key, &ipv6), ctxIPv4Key, &ipv4),
			func(m *mockZonefileWriter, wr io.Writer) error {
				if m.s.IPv4 != &ipv4 {
					t.Errorf("got %v instead of %v", m.s.IPv4, &ipv4)
				}
				if m.s.IPv6 != &ipv6 {
					t.Errorf("got %v instead of %v", m.s.IPv6, &ipv6)
				}
				return nil
			},
		},
	} {
		file, err := os.CreateTemp("", "zone-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file.Name())

		route := chi.NewRouter()
		route.Get("/", ZonefileWriteHandler(file.Name(), "example", &mockZonefileWriter{checkWrite: testCase.checkWrite}))

		w := httptest.NewRecorder()
		route.ServeHTTP(w, httptest.NewRequest("GET", "/", nil).WithContext(testCase.ctx))
		resp := w.Result()
		if resp.StatusCode != 200 {
			t.Errorf("status code is %v instead of %v", resp.StatusCode, 200)
		}

	}
}

func TestHttpRouter(t *testing.T) {
	route := chi.NewRouter()
	route.Mount("/", updaterHandler(updaterHandlerConfig{}))
}
