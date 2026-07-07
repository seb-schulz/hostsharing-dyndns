package main

import (
	"net/http"
)

// RejectBotsMiddleware short-circuits traffic that is obviously not a
// Fritz!Box DynDNS update. A request is let through only if ALL of:
// method is GET, path is "/", and the "user" query parameter is non-empty.
// Everything else gets an empty 403 before the argon2id validation runs.
func RejectBotsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			reject(w)
			return
		}
		if r.URL.Path != "/" {
			reject(w)
			return
		}
		if r.URL.Query().Get("user") == "" {
			reject(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func reject(w http.ResponseWriter) {
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(http.StatusForbidden)
}
