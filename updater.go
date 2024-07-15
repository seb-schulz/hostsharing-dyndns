package main

import (
	"crypto/subtle"
	"net/http"

	"golang.org/x/crypto/argon2"
)

type hashedPassword struct {
	key     []byte
	salt    []byte
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func (p *hashedPassword) isValid(origPasswd []byte) bool {
	key := argon2.IDKey(origPasswd, p.salt, p.time, p.memory, p.threads, p.keyLen)
	return subtle.ConstantTimeCompare(key, p.key) == 1
}

func PasswordValidationMiddleware(validate func(origPasswd []byte) bool) func(next http.Handler) http.Handler {
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
