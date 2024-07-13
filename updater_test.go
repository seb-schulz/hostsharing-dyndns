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

type updater struct {
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
	}
	if !ip6addr.Is6() {
		http.Error(w, "ip6addr is incorrect", http.StatusBadRequest)
	}

	log := httplog.LogEntry(r.Context())
	log.Debug("IP Adress", ipaddr.String(), ip6addr)

	fmt.Fprintln(w, "Ok")
}

func TestUpdaterHandler(t *testing.T) {
	route := chi.NewRouter()
	route.Mount("/", &updater{})

	for _, testCase := range []struct {
		input              *http.Request
		expectedStatusCode int
	}{
		{httptest.NewRequest("GET", "/", nil), 401},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.", nil), 200},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ipaddr=a.168.1.1", nil), 400},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ipaddr=192.168.1.1", nil), 200},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ip6addr=2001:db8::x", nil), 400},
		{httptest.NewRequest("GET", "/?user=test&passwd=.test.&ip6addr=2001:db8::1", nil), 200},
	} {
		w := httptest.NewRecorder()
		route.ServeHTTP(w, testCase.input)
		resp := w.Result()
		if resp.StatusCode != testCase.expectedStatusCode {
			t.Errorf("status code is %v instead of %v", resp.StatusCode, testCase.expectedStatusCode)
		}
	}
}
