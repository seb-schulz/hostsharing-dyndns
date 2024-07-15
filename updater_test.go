package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHashedPasswordValidation(t *testing.T) {
	for _, testCase := range []struct {
		h              hashedPassword
		password       []byte
		expectedResult bool
	}{
		{
			hashedPassword{[]byte{230, 104, 139, 86, 35, 176, 125, 179, 79, 26, 88, 17, 178, 50, 28, 214, 27, 165, 105, 84, 225, 141, 44, 123, 62, 196, 70, 127, 108, 203, 144, 225}, []byte("123"), 1, 1, 1, 32},
			[]byte(".test."), true,
		}, {
			hashedPassword{[]byte{110, 229, 10, 51, 153, 202, 41, 137, 248, 79, 231, 236, 127, 187, 80, 94, 249, 57, 166, 194, 156, 43, 72, 188, 139, 201, 240, 81, 164, 31, 152, 176}, []byte("abc"), 1, 1, 1, 32},
			[]byte(".test."), true,
		}, {
			hashedPassword{[]byte{230, 104, 139, 86, 35, 176, 125, 179, 79, 26, 88, 17, 178, 50, 28, 214, 27, 165, 105, 84, 225, 141, 44, 123, 62, 196, 70, 127, 108, 203, 144, 225}, []byte("123"), 1, 1, 1, 32},
			[]byte("..test.."), false,
		},
	} {
		if got := testCase.h.isValid(testCase.password); got != testCase.expectedResult {
			t.Errorf("test result not as expected: %v instead of %v", got, testCase.expectedResult)
		}
	}
}

func TestPasswordValidationMiddleware(t *testing.T) {

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
			httptest.NewRequest("GET", "/?passwd=.test.", nil),
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
			httptest.NewRequest("GET", "/?passwd=.test.", nil),
			func(t *testing.T, origPasswd []byte) bool {
				return false
			},
			401,
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

type ctxIPKey struct{ uint8 }

var ctxIPv4Key = ctxIPKey{uint8: 0}
var ctxIPv6Key = ctxIPKey{uint8: 1}

func IPValidationMiddleware(next http.Handler) http.Handler {
	parseAddrOrEmpty := func(ipStr string) (*netip.Addr, error) {
		if ipStr == "" {
			return nil, nil
		}
		r, err := netip.ParseAddr(ipStr)
		if err != nil {
			return nil, err
		}
		return &r, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		valid := true

		ipaddr, err := parseAddrOrEmpty(r.URL.Query().Get("ipaddr"))
		if err != nil {
			valid = false
			http.Error(w, "ipaddr is incorrect", http.StatusBadRequest)
		}
		if ipaddr != nil && !ipaddr.Is4() {
			valid = false
			http.Error(w, "ipaddr is incorrect", http.StatusBadRequest)
		}

		ip6addr, err := parseAddrOrEmpty(r.URL.Query().Get("ip6addr"))
		if err != nil {
			valid = false
			http.Error(w, "ip6addr is incorrect", http.StatusBadRequest)
		}
		if ip6addr != nil && !ip6addr.Is6() {
			valid = false
			http.Error(w, "ip6addr is incorrect", http.StatusBadRequest)
		}

		if valid && ipaddr != nil {
			ctx = context.WithValue(ctx, ctxIPv4Key, ipaddr)
		}
		if valid && ip6addr != nil {
			ctx = context.WithValue(ctx, ctxIPv6Key, ip6addr)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
