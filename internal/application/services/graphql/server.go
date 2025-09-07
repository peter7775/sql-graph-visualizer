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

package graphql

import (
	"errors"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/sirupsen/logrus"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/interfaces/graphql"
	"sql-graph-visualizer/internal/interfaces/graphql/generated"
)

// Server represents the GraphQL server
type Server struct {
	neo4jRepo ports.Neo4jPort
	config    *models.Config
	server    *http.Server
}

// NewServer creates a new GraphQL server
func NewServer(neo4jRepo ports.Neo4jPort, config *models.Config) *Server {
	return &Server{
		neo4jRepo: neo4jRepo,
		config:    config,
	}
}

// Start starts the GraphQL server
func (s *Server) Start(addr string) error {
	// Create resolver with dependencies
	resolver := &graphql.Resolver{
		Neo4jRepo: s.neo4jRepo,
		Config:    s.config,
	}

	// Create GraphQL handler
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Create HTTP mux
	mux := http.NewServeMux()

	// GraphQL endpoint
	mux.Handle("/graphql", srv)

	// GraphQL Playground
	mux.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))

	// Create HTTP server
	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logrus.Infof("GraphQL server starting on %s", addr)
	logrus.Infof("GraphQL Playground available at http://%s/playground", addr)

	return s.server.ListenAndServe()
}

// Stop stops the GraphQL server
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}

	logrus.Info("Stopping GraphQL server...")
	return s.server.Close()
}

// StartGraphQLServer is a convenience function to start the GraphQL server in a goroutine
func StartGraphQLServer(neo4jRepo ports.Neo4jPort, config *models.Config) *Server {
	server := NewServer(neo4jRepo, config)

	go func() {
		if err := server.Start("localhost:8081"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Errorf("GraphQL server failed: %v", err)
		}
	}()

	return server
}
