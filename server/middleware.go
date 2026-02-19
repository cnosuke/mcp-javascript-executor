package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

func withAuthMiddleware(next http.Handler, authToken string) http.Handler {
	// Pre-compute hash so ConstantTimeCompare always compares equal-length slices,
	// preventing token length from leaking via timing differences.
	var expectedHash [32]byte
	if authToken != "" {
		expectedHash = sha256.Sum256([]byte(authToken))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		token := extractBearerToken(r)
		tokenHash := sha256.Sum256([]byte(token))
		if subtle.ConstantTimeCompare(tokenHash[:], expectedHash[:]) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withOriginValidation(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(allowedOrigins) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		for _, allowed := range allowedOrigins {
			if origin == allowed {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
