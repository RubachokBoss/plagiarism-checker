package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/cors"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func RequestLogger(log zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = "unknown"
			}

			requestLog := log.With().
				Str("request_id", reqID).
				Logger()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			ctx := context.WithValue(r.Context(), "logger", requestLog)
			r = r.WithContext(ctx)

			defer func() {
				requestLog.Info().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("query", r.URL.RawQuery).
					Str("ip", r.RemoteAddr).
					Str("user_agent", r.UserAgent()).
					Int("status", ww.Status()).
					Int("bytes", ww.BytesWritten()).
					Dur("duration", time.Since(start)).
					Msg("HTTP request")
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func Recovery(log zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
					log.Error().
						Interface("recover", rvr).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("ip", r.RemoteAddr).
						Msg("Panic recovered")

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "Internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}

func NewCORS(allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders []string,
	allowCredentials bool, maxAge int) func(http.Handler) http.Handler {

	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		ExposedHeaders:   exposedHeaders,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	})
}

func GetLoggerFromContext(ctx context.Context) zerolog.Logger {
	if logger, ok := ctx.Value("logger").(zerolog.Logger); ok {
		return logger
	}
	return zerolog.Nop() // возвращаем пустой логгер если не найден
}
