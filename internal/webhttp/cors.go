package webhttp

import "net/http"

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
	MaxAge       string // e.g. "86400"
}

// DefaultCORSConfig returns a permissive CORS config suitable for development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type", "Accept"},
		MaxAge:       "86400",
	}
}

// CORS returns a middleware that handles CORS preflight and response headers.
func CORS(cfg CORSConfig) Middleware {
	origins := joinOrStar(cfg.AllowOrigins)
	methods := join(cfg.AllowMethods)
	headers := join(cfg.AllowHeaders)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			if cfg.MaxAge != "" {
				w.Header().Set("Access-Control-Max-Age", cfg.MaxAge)
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func joinOrStar(ss []string) string {
	if len(ss) == 0 {
		return "*"
	}
	return join(ss)
}

func join(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += ", " + s
	}
	return out
}
