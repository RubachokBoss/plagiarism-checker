package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

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
