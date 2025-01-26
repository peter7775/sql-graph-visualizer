package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	graphql "mysql-graph-visualizer/internal/application/services/graphql/generated"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	graphqlModels "mysql-graph-visualizer/internal/domain/models/graphql"
	"mysql-graph-visualizer/internal/domain/repositories/config"
	"mysql-graph-visualizer/internal/infrastructure/middleware"

	"github.com/99designs/gqlgen/graphql/playground"
)



// GetConfig returns the configuration as JSON
func GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := config.Load()
	if err != nil {
		http.Error(w, "Chyba při načítání konfigurace", http.StatusInternalServerError)
		log.Printf("Chyba při načítání konfigurace: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Chyba při kódování JSON", http.StatusInternalServerError)
		log.Printf("Chyba při kódování JSON: %v", err)
		return
	}
}

// Spuštění GraphQL serveru
func StartGraphQLServer() {

	// Přidání CORS middleware
	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}

	http.Handle("/graphql", middleware.NewCORSHandler(corsOptions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolver := &Resolver{}
		nodes, err := resolver.Query().Nodes(r.Context())
		if err != nil {
			http.Error(w, "Chyba při získávání uzlů", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})))
	http.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))
	http.Handle("/config", middleware.NewCORSHandler(corsOptions)(http.HandlerFunc(GetConfig)))

	log.Println("Spouštím GraphQL server na http://localhost:8080/")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Chyba při spuštění serveru: %v", err)
	}
}

// Define the Resolver type locally
type Resolver struct{}

// Implement the Query method
func (r *Resolver) Query() graphql.QueryResolver {
	return &queryResolver{r}
}

// Implement the Nodes resolver
func (r *queryResolver) Nodes(ctx context.Context) ([]*graphqlModels.Node, error) {
	graphAgg := graph.NewGraphAggregate("")
	nodes := graphAgg.GetNodes()

	// Convert entities.Node to graphqlModels.Node
	var gqlNodes []*graphqlModels.Node
	for _, node := range nodes {
		// Convert map[string]interface{} to *graphqlModels.Properties
		var props *graphqlModels.Properties
		if node.Properties != nil {
			props = &graphqlModels.Properties{
				Key:   node.Properties["key"].(*string),
				Value: node.Properties["value"].(*string),
			}
		}

		gqlNode := &graphqlModels.Node{
			ID:         node.ID,
			Label:      node.Type,
			Properties: props,
		}
		gqlNodes = append(gqlNodes, gqlNode)
	}

	return gqlNodes, nil
}

type queryResolver struct {
	*Resolver
}
