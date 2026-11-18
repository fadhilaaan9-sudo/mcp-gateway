package middlewares

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Auth adalah penjaga pintu: hanya yang membawa token yang benar
// yang boleh masuk ke /messages. Token dikirim lewat header:
// Authorization: Bearer <token>
// /health boleh lewat tanpa token, supaya gampang cek server hidup.
func Auth(tokens map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if token == "" {
				writeError(w, http.StatusUnauthorized, "token tidak ada")
				return
			}

			for _, allowed := range tokens {
				if token == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeError(w, http.StatusUnauthorized, "token tidak valid")
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
