package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os"

	"github.com/go-chi/chi/v5"
)

type ctxIPKey struct{ uint8 }

type passwordValidator func(origPasswd []byte) bool

type updaterHandlerConfig struct {
	User     string
	Password struct {
		Key     []byte
		Salt    []byte
		Time    uint32
		Memory  uint32
		Threads uint8
		KeyLen  uint32
	}
	Filename      string
	DomainSubpart string
}

var ctxIPv4Key = ctxIPKey{uint8: 0}
var ctxIPv6Key = ctxIPKey{uint8: 1}

func PasswordValidationMiddleware(validate passwordValidator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			passwd := r.URL.Query().Get("passwd")

			if passwd == "" {
				http.Error(w, "user or password wrong", http.StatusUnauthorized)
				return
			}

			if !validate([]byte(passwd)) {
				http.Error(w, "user or password wrong", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func UserValidationMiddleware(user string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			unverifiedUser := r.URL.Query().Get("user")

			if unverifiedUser == "" {
				http.Error(w, "user or password wrong", http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(user), []byte(unverifiedUser)) != 1 {
				http.Error(w, "user or password wrong", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

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

func ZonefileWriteHandler(filename string, domainSubpart string, z zoneFileWriter) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ipaddr, _ := r.Context().Value(ctxIPv4Key).(*netip.Addr)
		ipv6addr, _ := r.Context().Value(ctxIPv6Key).(*netip.Addr)

		z.Set(subdomain{
			Subpart: domainSubpart,
			TTL:     60,
			IPv4:    ipaddr,
			IPv6:    ipv6addr,
		})

		f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("cannot write file: %v", err)
		}
		if err := z.Write(f); err != nil {
			log.Println("Cannot write zonefile")
		}
		fmt.Fprintln(w, "Ok")
	}
}

func updaterHandler(c updaterHandlerConfig) http.Handler {
	route := chi.NewRouter()
	route.Use(UserValidationMiddleware(c.User))
	route.Use(PasswordValidationMiddleware(argonPasswordValidator(c.Password.Key, c.Password.Salt, c.Password.Time, c.Password.Memory, c.Password.Threads, c.Password.KeyLen)))
	route.Use(IPValidationMiddleware)
	route.Get("/", ZonefileWriteHandler(c.Filename, c.DomainSubpart, newZonefile()))
	return route
}
