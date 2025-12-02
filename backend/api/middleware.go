package api

import (
	"net/http"
	"strings"

	"github.com/nnk/97-aic/backend/config"
)

// CORSMiddleware добавляет CORS заголовки
func CORSMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if cfg.IsCORSAllowed(origin) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LimitBodyMiddleware ограничивает размер тела запроса
func LimitBodyMiddleware(maxBytes int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
		}
		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware добавляет ID запроса
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Можно использовать X-Request-ID из заголовка или генерировать
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

// generateRequestID генерирует уникальный ID запроса
func generateRequestID() string {
	// Простая реализация на основе времени
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.TrimPrefix(
				strings.TrimSuffix(
					http.TimeFormat,
					" MST",
				),
				"Mon, ",
			),
			" ", "-",
		),
		":", "",
	)
}
