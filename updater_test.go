package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
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

type passwordValidator interface {
	isValid(origPasswd []byte) bool
}

type updater struct {
	user              string
	passwordvalidator passwordValidator
}

// func newUpdater() http.Handler {
// 	route := chi.NewRouter()

// }

func parseAddrOrEmpty(ipStr string, ipv6 bool) (netip.Addr, error) {
	if ipStr == "" && !ipv6 {
		return netip.MustParseAddr("0.0.0.0"), nil
	}
	if ipStr == "" && ipv6 {
		return netip.MustParseAddr("::"), nil
	}
	return netip.ParseAddr(ipStr)
}

func (u *updater) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	passwd := r.URL.Query().Get("passwd")

	if user == "" || passwd == "" {
		http.Error(w, "user or password wrong", http.StatusUnauthorized)
		return
	}

	if !u.passwordvalidator.isValid([]byte(passwd)) {
		http.Error(w, "user or password wrong", http.StatusUnauthorized)
		return
	}

	ipaddr, err := parseAddrOrEmpty(r.URL.Query().Get("ipaddr"), false)
	if err != nil {
		http.Error(w, "ipaddr is incorrect", http.StatusBadRequest)
	}
	if !ipaddr.Is4() {
		http.Error(w, "ipaddr is incorrect", http.StatusBadRequest)
	}

	ip6addr, err := parseAddrOrEmpty(r.URL.Query().Get("ip6addr"), true)
	if err != nil {
		http.Error(w, "ip6addr is incorrect", http.StatusBadRequest)
		return
	}
	if !ip6addr.Is6() {
		http.Error(w, "ip6addr is incorrect", http.StatusBadRequest)
		return
	}

	log := httplog.LogEntry(r.Context())
	log.Debug("IP Adress", ipaddr.String(), ip6addr)

	fmt.Fprintln(w, "Ok")
}

type mockPasswordValidator struct {
	called  uint
	checker func(origPasswd []byte) bool
}

func (m *mockPasswordValidator) isValid(origPasswd []byte) bool {
	m.called++
	return m.checker(origPasswd)
}

func TestUpdaterHandler(t *testing.T) {

	handler := updater{}
	route := chi.NewRouter()
	route.Mount("/", &handler)

	for _, testCase := range []struct {
		input                 *http.Request
		mockPasswordValidator mockPasswordValidator
		expectedStatusCode    int
	}{
		{httptest.NewRequest("GET", "/", nil), mockPasswordValidator{checker: func(origPasswd []byte) bool {
			t.FailNow()
			return false
		}}, 401},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.", nil), mockPasswordValidator{checker: func(origPasswd []byte) bool {
			if string(origPasswd) != ".test." {
				t.Errorf("unexpected password")
			}
			return true
		}}, 200},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ipaddr=a.168.1.1", nil), mockPasswordValidator{checker: func(origPasswd []byte) bool {
			if string(origPasswd) != ".test." {
				t.Errorf("unexpected password")
				return false
			}
			return true
		}}, 400},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ipaddr=192.168.1.1", nil), mockPasswordValidator{checker: func(origPasswd []byte) bool {
			if string(origPasswd) != ".test." {
				t.Errorf("unexpected password")
				return false
			}
			return true
		}}, 200},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ip6addr=2001:db8::x", nil), mockPasswordValidator{checker: func(origPasswd []byte) bool {
			if string(origPasswd) != ".test." {
				t.Errorf("unexpected password")
				return false
			}
			return true
		}}, 400},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ip6addr=2001:db8::1", nil), mockPasswordValidator{checker: func(origPasswd []byte) bool {
			if string(origPasswd) != ".test." {
				t.Errorf("unexpected password")
				return false
			}
			return true
		}}, 200},
	} {
		handler.passwordvalidator = &testCase.mockPasswordValidator

		w := httptest.NewRecorder()
		route.ServeHTTP(w, testCase.input)
		resp := w.Result()
		if resp.StatusCode != testCase.expectedStatusCode {
			t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatusCode)
		}
	}
}
