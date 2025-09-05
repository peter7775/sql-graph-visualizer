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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	neo4jDriver "github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/sirupsen/logrus"

	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/application/services/graphql/server"
	"mysql-graph-visualizer/internal/application/services/transform"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	"mysql-graph-visualizer/internal/domain/models"
	"mysql-graph-visualizer/internal/domain/repositories/config"
	"mysql-graph-visualizer/internal/domain/repositories/configrule"
	"mysql-graph-visualizer/internal/infrastructure/middleware"
	mysqlrepo "mysql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"mysql-graph-visualizer/internal/infrastructure/persistence/neo4j"
)

var addr = "127.0.0.1:3000"

func main() {
	ctx := context.Background()

	logrus.Infof("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}
	logrus.Infof("Configuration loaded: %+v", cfg)

	logrus.Infof("cfg.Neo4jURI type: %T, value: %s", cfg.Neo4j.URI, cfg.Neo4j.URI)
	logrus.Infof("cfg.Neo4jUser type: %T, value: %s", cfg.Neo4j.User, cfg.Neo4j.User)
	logrus.Infof("cfg.Neo4jPassword type: %T, value: %s", cfg.Neo4j.Password, cfg.Neo4j.Password)

	logrus.Infof("Initializing MySQL connection...")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)
	logrus.Infof("DSN: %s", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logrus.Fatalf("Failed to connect to MySQL: %v", err)
	}
	logrus.Infof("MySQL connection successful")

	mysqlRepo := mysqlrepo.NewMySQLRepository(db)
	defer mysqlRepo.Close()

	logrus.Infof("Initializing Neo4j connection...")
	logrus.Infof("cfg.Neo4jURI type: %T, value: %s", cfg.Neo4j.URI, cfg.Neo4j.URI)
	neo4jRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		logrus.Fatalf("Failed to create Neo4j repository: %v", err)
	}
	logrus.Infof("Neo4j connection successful")
	defer neo4jRepo.Close()

	logrus.Infof("Deleting all data in Neo4j...")
	session := neo4jRepo.NewSession(neo4jDriver.SessionConfig{})
	defer session.Close()

	_, err = session.Run("MATCH (n) DETACH DELETE n", nil)
	if err != nil {
 	   logrus.Fatalf("Error deleting data in Neo4j: %v", err)
	}
	logrus.Infof("All data in Neo4j deleted")

	logrus.Infof("Initializing services...")
	transformService := transform.NewTransformService(mysqlRepo, neo4jRepo, configrule.NewRuleRepository())
	logrus.Infof("Services initialized")

	go server.StartGraphQLServer(neo4jRepo)

	logrus.Infof("Starting data transformation...")
	if err := transformService.TransformAndStore(ctx); err != nil {
		logrus.Fatalf("Failed to transform and store data: %v", err)
	}
	logrus.Infof("Data transformation successful")

	logrus.Infof("Starting server...")
	startVisualizationServer(neo4jRepo, cfg)

	router := mux.NewRouter()

	router.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	})

	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(router)

	server := &http.Server{
		Handler: handler,
		Addr:    "localhost:8080",
	}

	logrus.Infof("Starting server on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

func startVisualizationServer(neo4jRepo ports.Neo4jPort, cfg *models.Config) *http.Server {
	logrus.Infof("Starting visualization server")
	mux := http.NewServeMux()

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to /config endpoint")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		configResponse := map[string]any{
			"neo4j": map[string]string{
				"uri":      cfg.Neo4j.URI,
				"username": cfg.Neo4j.User,
				"password": cfg.Neo4j.Password,
			},
		}
		
		if err := json.NewEncoder(w).Encode(configResponse); err != nil {
			logrus.Errorf("Error encoding config response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logrus.Infof("Config response sent successfully")
	})

	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to API endpoint /api/graph")

		graphInterface, err := neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
		if err != nil {
			logrus.Errorf("Error retrieving data: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		g, ok := graphInterface.(*graph.GraphAggregate)
		if !ok {
			logrus.Warnf("Invalid graph type")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		response := struct {
			Nodes         []map[string]any `json:"nodes"`
			Relationships []map[string]any `json:"relationships"`
		}{
			Nodes:         make([]map[string]any, 0),
			Relationships: make([]map[string]any, 0),
		}

		for _, node := range g.GetNodes() {
			nodeData := map[string]any{
				"id":         node.ID,
				"label":      node.Type,
				"properties": node.Properties,
			}
			response.Nodes = append(response.Nodes, nodeData)
			logrus.Infof("Adding node: %v", nodeData)
		}

		for _, rel := range g.GetRelationships() {
			relData := map[string]any{
				"from":       rel.SourceNode.ID,
				"to":         rel.TargetNode.ID,
				"type":       rel.Type,
				"properties": rel.Properties,
			}
			response.Relationships = append(response.Relationships, relData)
			logrus.Infof("Adding relationship: %v", relData)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		logrus.Infof("Sending response: %d nodes, %d relationships", len(response.Nodes), len(response.Relationships))

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error serializing response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	webRoot := filepath.Join(findProjectRoot(), "internal", "interfaces", "web")
	logrus.Infof("Using web root: %s", webRoot)

	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to main page")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Warnf("Port %s is occupied: %v", addr, err)
		exec.Command("fuser", "-k", "3000/tcp").Run()
		time.Sleep(time.Second)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			logrus.Fatalf("Cannot create listener: %v", err)
		}
	}
	logrus.Infof("Listener created on %s", addr)

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		logrus.Warnf("Starting server on %s", addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server terminated with error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logrus.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Error shutting down server: %v", err)
	}
	logrus.Println("Server successfully shut down")

	logrus.Infof("Visualization is available at http://localhost:3000")
	return server
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Cannot get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			logrus.Fatalf("Cannot find project root directory")
			return ""
		}
		wd = parent
	}
}

func init() {
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
}
