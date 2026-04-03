package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Content-Length", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

func CORS(cfg *CORSConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultCORSConfig()
	}

	allowedOrigins := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowedOrigins[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if len(cfg.AllowedOrigins) > 0 && cfg.AllowedOrigins[0] != "*" {
				if origin != "" && !allowedOrigins[origin] {
					next.ServeHTTP(w, r)
					return
				}
			}

			if cfg.AllowedOrigins[0] == "*" && !cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
			w.Header().Set("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}