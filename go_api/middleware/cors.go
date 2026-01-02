package middleware

import (
	"net/http"
)

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// For development: allow all origins (simpler and works for localhost)
		// In production, you should whitelist specific origins
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Key")
		// Removed Access-Control-Allow-Credentials since we use JWT tokens in Authorization header
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight OPTIONS request - MUST return before calling next handler
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
