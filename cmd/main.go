/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v4/neo4j"

	"mysql-graph-visualizer/internal/application/services/migration"
	"mysql-graph-visualizer/internal/application/services/transform"
	"mysql-graph-visualizer/internal/application/services/visualization"
	"mysql-graph-visualizer/internal/config"
	"mysql-graph-visualizer/internal/domain/repositories"
	"mysql-graph-visualizer/internal/domain/valueobjects"
	mysqlrepo "mysql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"mysql-graph-visualizer/internal/infrastructure/persistence/neo4j"
	"mysql-graph-visualizer/internal/interfaces/http/handlers"
)

func main() {
	ctx := context.Background()

	// Inicializace konfigurace
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Nepodařilo se načíst konfiguraci: %v", err)
	}

	// Initialize MySQL connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Initialize repositories
	mysqlRepo := mysqlrepo.NewMySQLRepository(db)
	defer mysqlRepo.Close()

	neo4jDriver, err := neo4jdriver.NewDriver(cfg.Neo4j.URI, neo4jdriver.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	neo4jRepo := neo4j.NewNeo4jRepository(neo4jDriver)
	defer neo4jRepo.Close()

	// Initialize services
	transformService := transform.NewTransformService(mysqlRepo, neo4jRepo, repositories.NewInMemoryRuleRepository())
	migrationService := migration.NewMigrationService(mysqlRepo, neo4jRepo, transformService)
	visualizationService := visualization.NewVisualizationService(neo4jRepo)

	// Setup router
	router := mux.NewRouter()
	setupRoutes(router, cfg, visualizationService)

	// Run data transformation
	if err := migrationService.MigrateData(ctx, valueobjects.TransformConfig{}); err != nil {
		log.Fatalf("Failed to transform and store data: %v", err)
	}

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	log.Printf("Server starting on port %d", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func setupRoutes(router *mux.Router, config *config.Config, visualizationService *visualization.VisualizationService) {
	handler := handlers.NewVisualizationHandler(visualizationService)

	router.HandleFunc("/api/visualization/config", handler.GetConfig).Methods("GET")
	router.HandleFunc("/visualization", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/visualization.html")
	}).Methods("GET")
}
