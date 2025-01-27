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
	"mysql-graph-visualizer/internal/domain/repositories/config"
	"mysql-graph-visualizer/internal/domain/repositories/configrule"
	"mysql-graph-visualizer/internal/infrastructure/middleware"
	mysqlrepo "mysql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"mysql-graph-visualizer/internal/infrastructure/persistence/neo4j"
)

var addr = "127.0.0.1:3000"

func main() {
	ctx := context.Background()

	logrus.Infof("Načítám konfiguraci...")
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Nepodařilo se načíst konfiguraci: %v", err)
	}
	logrus.Infof("Konfigurace načtena: %+v", cfg)

	logrus.Infof("Typ cfg.Neo4jURI: %T, hodnota: %s", cfg.Neo4j.URI, cfg.Neo4j.URI)
	logrus.Infof("Typ cfg.Neo4jUser: %T, hodnota: %s", cfg.Neo4j.User, cfg.Neo4j.User)
	logrus.Infof("Typ cfg.Neo4jPassword: %T, hodnota: %s", cfg.Neo4j.Password, cfg.Neo4j.Password)

	// Initialize MySQL connection
	logrus.Infof("Inicializuji připojení k MySQL...")
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
	logrus.Infof("Připojení k MySQL úspěšné")

	// Start the GraphQL server
	server.StartGraphQLServer()

	// Initialize repositories
	mysqlRepo := mysqlrepo.NewMySQLRepository(db)
	defer mysqlRepo.Close()

	// Initialize Neo4j connection
	logrus.Infof("Inicializuji připojení k Neo4j...")
	logrus.Infof("Typ cfg.Neo4jURI: %T, hodnota: %s", cfg.Neo4j.URI, cfg.Neo4j.URI)
	neo4jRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		logrus.Fatalf("Failed to create Neo4j repository: %v", err)
	}
	logrus.Infof("Připojení k Neo4j úspěšné")
	defer neo4jRepo.Close()

	// Smazání všech dat v Neo4j
	logrus.Infof("Mažu všechna data v Neo4j...")
	session := neo4jRepo.NewSession(neo4jDriver.SessionConfig{})
	defer session.Close()

	_, err = session.Run("MATCH (n) DETACH DELETE n", nil)
	if err != nil {
 	   logrus.Fatalf("Chyba při mazání dat v Neo4j: %v", err)
	}
	logrus.Infof("Všechna data v Neo4j byla smazána")

	// Initialize services
	logrus.Infof("Inicializuji služby...")
	transformService := transform.NewTransformService(mysqlRepo, neo4jRepo, configrule.NewRuleRepository())
	logrus.Infof("Služby inicializovány")

	// Run data transformation
	logrus.Infof("Spouštím transformaci dat...")
	if err := transformService.TransformAndStore(ctx); err != nil {
		logrus.Fatalf("Failed to transform and store data: %v", err)
	}
	logrus.Infof("Transformace dat úspěšná")

	// Start server
	logrus.Infof("Spouštím server...")
	startVisualizationServer(neo4jRepo)

	// Initialize the router
	router := mux.NewRouter()

	// Define your routes
	router.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	})

	// Add CORS middleware
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

	logrus.Infof("Spouštím server na %s", addr)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Nepodařilo se spustit server: %v", err)
	}
}

func startVisualizationServer(neo4jRepo ports.Neo4jPort) *http.Server {
	logrus.Infof("Začínám spouštět vizualizační server")
	mux := http.NewServeMux()

	// API endpoint pro data grafu
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Požadavek na API endpoint /api/graph")

		// Získáme celý graf
		graphInterface, err := neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
		if err != nil {
			logrus.Errorf("Chyba při získávání dat: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		g, ok := graphInterface.(*graph.GraphAggregate)
		if !ok {
			logrus.Warnf("Neplatný typ grafu")
			http.Error(w, "Interní chyba serveru", http.StatusInternalServerError)
			return
		}

		// Převedeme na JSON
		response := struct {
			Nodes         []map[string]interface{} `json:"nodes"`
			Relationships []map[string]interface{} `json:"relationships"`
		}{
			Nodes:         make([]map[string]interface{}, 0),
			Relationships: make([]map[string]interface{}, 0),
		}

		// Přidáme uzly
		for _, node := range g.GetNodes() {
			nodeData := map[string]interface{}{
				"id":         node.ID,
				"label":      node.Type,
				"properties": node.Properties,
			}
			response.Nodes = append(response.Nodes, nodeData)
			logrus.Infof("Přidávám uzel: %v", nodeData)
		}

		// Přidáme vztahy
		for _, rel := range g.GetRelationships() {
			relData := map[string]interface{}{
				"from":       rel.SourceNode.ID,
				"to":         rel.TargetNode.ID,
				"type":       rel.Type,
				"properties": rel.Properties,
			}
			response.Relationships = append(response.Relationships, relData)
			logrus.Infof("Přidávám vztah: %v", relData)
		}

		// Nastavíme hlavičky
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Vypíšeme data pro debug
		logrus.Infof("Odesílám odpověď: %d uzlů, %d vztahů", len(response.Nodes), len(response.Relationships))

		// Odešleme odpověď
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Chyba při serializaci odpovědi: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	webRoot := filepath.Join(findProjectRoot(), "internal", "interfaces", "web")
	logrus.Infof("Používám web root: %s", webRoot)

	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Servírujeme HTML stránku
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Požadavek na hlavní stránku")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	// Vytvoříme listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Warnf("Port %s je obsazený: %v", addr, err)
		// Pokusíme se ukončit proces na daném portu
		exec.Command("fuser", "-k", "3000/tcp").Run()
		time.Sleep(time.Second)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			logrus.Fatalf("Nelze vytvořit listener: %v", err)
		}
	}
	logrus.Infof("Listener vytvořen na %s", addr)

	// Spustíme server
	server := &http.Server{
		Handler: mux,
	}

	go func() {
		logrus.Warnf("Spouštím server na %s", addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server ukončen s chybou: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logrus.Println("Ukončuji server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Chyba při ukončování serveru: %v", err)
	}
	logrus.Println("Server úspěšně ukončen")

	logrus.Infof("Vizualizace je dostupná na http://localhost:3000")
	return server
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Nelze získat pracovní adresář: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			logrus.Fatalf("Nelze najít kořenový adresář projektu")
			return ""
		}
		wd = parent
	}
}

func init() {
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel) // Default level
	} else {
		logrus.SetLevel(level)
	}
}
