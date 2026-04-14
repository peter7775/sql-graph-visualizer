/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */

// Package middleware provides HTTP middleware implementations.
package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORSOptions represents CORS configuration options.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// NewCORSHandler creates a new CORS middleware handler.
func NewCORSHandler(options CORSOptions) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   options.AllowedOrigins,
		AllowedMethods:   options.AllowedMethods,
		AllowedHeaders:   options.AllowedHeaders,
		AllowCredentials: options.AllowCredentials,
	})
	return c.Handler
}
