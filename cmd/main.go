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
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/application/services/transform"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	"mysql-graph-visualizer/internal/domain/repositories/config"
	"mysql-graph-visualizer/internal/domain/repositories/configrule"
	mysqlrepo "mysql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"mysql-graph-visualizer/internal/infrastructure/persistence/neo4j"
)

var addr = "127.0.0.1:3000"

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

	neo4jRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		log.Fatalf("Failed to create Neo4j repository: %v", err)
	}
	defer neo4jRepo.Close()

	// Initialize services
	transformService := transform.NewTransformService(mysqlRepo, neo4jRepo, configrule.NewRuleRepository())

	// Run data transformation
	if err := transformService.TransformAndStore(ctx); err != nil {
		log.Fatalf("Failed to transform and store data: %v", err)
	}

	// Start server
	startVisualizationServer(neo4jRepo)
}

func startVisualizationServer(neo4jRepo ports.Neo4jPort) *http.Server {
	log.Printf("Začínám spouštět vizualizační server")
	mux := http.NewServeMux()

	// API endpoint pro data grafu
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Požadavek na API endpoint /api/graph")

		// Získáme celý graf
		graphInterface, err := neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
		if err != nil {
			log.Printf("Chyba při získávání dat: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		g, ok := graphInterface.(*graph.GraphAggregate)
		if !ok {
			log.Printf("Neplatný typ grafu")
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
			log.Printf("Přidávám uzel: %v", nodeData)
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
			log.Printf("Přidávám vztah: %v", relData)
		}

		// Nastavíme hlavičky
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Vypíšeme data pro debug
		log.Printf("Odesílám odpověď: %d uzlů, %d vztahů", len(response.Nodes), len(response.Relationships))

		// Odešleme odpověď
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Chyba při serializaci odpovědi: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	webRoot := filepath.Join(findProjectRoot(), "internal", "interfaces", "web")
	log.Printf("Používám web root: %s", webRoot)

	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Servírujeme HTML stránku
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Požadavek na hlavní stránku")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	// Vytvoříme listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Port %s je obsazený: %v", addr, err)
		// Pokusíme se ukončit proces na daném portu
		exec.Command("fuser", "-k", "3000/tcp").Run()
		time.Sleep(time.Second)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Nelze vytvořit listener: %v", err)
		}
	}
	log.Printf("Listener vytvořen na %s", addr)

	// Spustíme server
	server := &http.Server{
		Handler: mux,
	}

	go func() {
		log.Printf("Spouštím server na %s", addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server ukončen s chybou: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Ukončuji server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Chyba při ukončování serveru: %v", err)
	}
	log.Println("Server úspěšně ukončen")

	log.Printf("Vizualizace je dostupná na http://localhost:3000")
	return server
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Nelze získat pracovní adresář: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			log.Fatalf("Nelze najít kořenový adresář projektu")
			return ""
		}
		wd = parent
	}
}
