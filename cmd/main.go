package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/peter7775/alevisualizer/internal/config"
	"github.com/peter7775/alevisualizer/internal/infrastructure"
	"github.com/peter7775/alevisualizer/internal/interfaces"
	"github.com/peter7775/alevisualizer/internal/interfaces/http/handlers"
	"github.com/peter7775/alevisualizer/internal/services"
)

func main() {
	// Initialize configuration
	config, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize repositories
	mysqlRepo, err := infrastructure.NewMySQLRepository(config.MySQL)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer mysqlRepo.Close()

	neo4jRepo, err := infrastructure.NewNeo4jRepository(config.Neo4j)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer neo4jRepo.Close()

	// Initialize services
	dataTransformService := services.NewDataTransformService(mysqlRepo, neo4jRepo)
	visualizationService := services.NewVisualizationService(neo4jRepo)

	// Setup router
	router := mux.NewRouter()
	setupRoutes(router, config, visualizationService)

	// Start HTTP server for visualization
	server := interfaces.NewHTTPServer(config.Server, router)

	// Run data transformation
	if err := dataTransformService.TransformAndStore(); err != nil {
		log.Fatalf("Failed to transform and store data: %v", err)
	}

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func setupRoutes(router *mux.Router, config *config.Config, visualizationService *services.VisualizationService) {
	vizHandler := handlers.NewVisualizationHandler(visualizationService)

	router.HandleFunc("/api/visualization/config", vizHandler.GetConfig).Methods("GET")
	router.HandleFunc("/visualization", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/visualization.html")
	}).Methods("GET")
}
