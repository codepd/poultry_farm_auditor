package middleware

import (
	"net/http"
	"strings"
)

// CORS returns a middleware that allows requests from whitelisted origins.
// If allowedOrigins is nil or empty, mirrors the request Origin (permissive, for dev).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]bool)
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowedSet[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				if len(allowedSet) > 0 {
					if allowedSet[origin] {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
					// If not in whitelist, do not set Access-Control-Allow-Origin (browser will block)
				} else {
					// No whitelist = permissive (dev): mirror origin
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			} else if len(allowedSet) == 0 {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Key")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
