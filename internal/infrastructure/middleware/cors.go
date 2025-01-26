package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORSOptions defines the options for the CORS middleware.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// NewCORSHandler creates a new CORS handler with the specified options.
func NewCORSHandler(options CORSOptions) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   options.AllowedOrigins,
		AllowedMethods:   options.AllowedMethods,
		AllowedHeaders:   options.AllowedHeaders,
		AllowCredentials: options.AllowCredentials,
	})
	return c.Handler
}
