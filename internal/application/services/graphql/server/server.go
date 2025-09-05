/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"mysql-graph-visualizer/internal/application/ports"
	graphql "mysql-graph-visualizer/internal/application/services/graphql/generated"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	graphqlModels "mysql-graph-visualizer/internal/domain/models/graphql"
	"mysql-graph-visualizer/internal/domain/repositories/config"
	"mysql-graph-visualizer/internal/infrastructure/middleware"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/sirupsen/logrus"
)

// GetConfig returns the configuration as JSON
func GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := config.Load()
	if err != nil {
		http.Error(w, "Error loading configuration", http.StatusInternalServerError)
		log.Printf("Error loading configuration: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
		return
	}
}

// Start GraphQL server
func StartGraphQLServer(neo4jPort ports.Neo4jPort) {
	// Add CORS middleware
	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}

	// Create GraphQL server
	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: &Resolver{neo4jPort: neo4jPort},
	}))

	// Add CORS middleware for GraphQL endpoint
	http.Handle("/graphql", middleware.NewCORSHandler(corsOptions)(srv))

	// Add GraphQL playground
	http.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))

	// Add configuration endpoint
	http.Handle("/config", middleware.NewCORSHandler(corsOptions)(http.HandlerFunc(GetConfig)))

	log.Println("Starting GraphQL server on http://localhost:8080/")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// Define the Resolver type locally
type Resolver struct {
	neo4jPort ports.Neo4jPort
}

// Implement the Query method
func (r *Resolver) Query() graphql.QueryResolver {
	return &queryResolver{r, r.neo4jPort}
}

// Implement the Nodes resolver
func (r *queryResolver) Nodes(ctx context.Context) ([]*graphqlModels.Node, error) {
	if r.neo4jPort == nil {
		return nil, fmt.Errorf("neo4jPort is not initialized")
	}

	// Load nodes from Neo4j
	phpActionNodes, err := r.neo4jPort.FetchNodes("PHPAction")
	if err != nil {
		return nil, fmt.Errorf("error loading PHPAction nodes from Neo4j: %v", err)
	}

	// Create GraphAggregate and add nodes
	graphAgg := graph.NewGraphAggregate("")
	for _, nodeData := range phpActionNodes {
		if err := graphAgg.AddNode("PHPAction", nodeData); err != nil {
			return nil, fmt.Errorf("error adding node to GraphAggregate: %v", err)
		}
	}

	// Convert entities.Node to graphqlModels.Node
	var gqlNodes []*graphqlModels.Node
	for _, node := range graphAgg.GetNodes() {
		// Convert map[string]any to *graphqlModels.Properties
		var props *graphqlModels.Properties
		if node.Properties != nil {
			key, keyOk := node.Properties["key"].(string)
			value, valueOk := node.Properties["value"].(string)
			if keyOk && valueOk {
				props = &graphqlModels.Properties{
					Key:   &key,
					Value: &value,
				}
			} else {
				logrus.Warnf("Error converting node properties: %+v", node.Properties)
			}
		}

		gqlNode := &graphqlModels.Node{
			ID:         node.ID,
			Label:      node.Type,
			Properties: props,
		}
		logrus.Infof("Converting node for GraphQL: ID=%s, Label=%s", node.ID, node.Type)
		gqlNodes = append(gqlNodes, gqlNode)
	}

	return gqlNodes, nil
}

type queryResolver struct {
	*Resolver
	neo4jPort ports.Neo4jPort
}
