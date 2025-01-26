package integration

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
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	// Adjust the actual import path as needed
	"mysql-graph-visualizer/internal/application/ports"
	server "mysql-graph-visualizer/internal/application/services/graphql/server"
	transformService "mysql-graph-visualizer/internal/application/services/transform"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	transformAggregates "mysql-graph-visualizer/internal/domain/aggregates/transform"
	transformObjects "mysql-graph-visualizer/internal/domain/valueobjects/transform"
	"mysql-graph-visualizer/internal/infrastructure/middleware"

	"mysql-graph-visualizer/internal/config"

	"mysql-graph-visualizer/internal/domain/repositories/neo4j"

	_ "github.com/go-sql-driver/mysql"
	// Use an alias for the Neo4j driver import

	"mysql-graph-visualizer/internal/domain/aggregates/serialization"

	neo4jDriver "github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type mockRuleRepo struct{}

func (m *mockRuleRepo) GetAllRules(ctx context.Context) ([]*transformAggregates.TransformRuleAggregate, error) {
	return []*transformAggregates.TransformRuleAggregate{
		{
			Rule: transformObjects.TransformRule{
				Name:       "php_actions_to_nodes",
				RuleType:   transformObjects.NodeRule,
				SourceSQL:  "SELECT * FROM alex_uzly au WHERE au.id_typu = 17",
				TargetType: "NodePHPAction",
				FieldMappings: map[string]string{
					"id_typu": "id_typu",
					"infix":   "infix",
					"nazev":   "name",
					"prefix":  "prefix",
				},
			},
		},
		{
			Rule: transformObjects.TransformRule{
				Name:       "php_actions",
				RuleType:   transformObjects.NodeRule,
				SourceSQL:  "SELECT * FROM alex_uzly_php_action au JOIN alex_uzly aupa ON au.id_node = aupa.id",
				TargetType: "PHPAction",
				FieldMappings: map[string]string{
					"id_node": "id_node",
				},
			},
		},
		//{
		//Rule: transformObjects.TransformRule{
		//Name:          "php_action_relationship",
		//RuleType:      transformObjects.RelationshipRule,
		//RelationType:  "AKCE",
		//Direction:     transformObjects.Outgoing,
		//SourceNode:    &transformObjects.NodeMapping{Type: "PHPAction", Key: "id", TargetField: "id"},
		//TargetNode:    &transformObjects.NodeMapping{Type: "NodePHPAction", Key: "id", TargetField: "id"},
		//FieldMappings: map[string]string{"id": "id"},
		//},
		//},
	}, nil
}

func (m *mockRuleRepo) SaveRule(ctx context.Context, rule *transformAggregates.TransformRuleAggregate) error {
	return nil
}

func (m *mockRuleRepo) DeleteRule(ctx context.Context, ruleID string) error {
	return nil
}

func (m *mockRuleRepo) UpdateRulePriority(ctx context.Context, ruleID string, priority int) error {
	return nil
}

type realMySQLRepo struct {
	db *sql.DB
}

func setupMySQLConnection() (*sql.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("Chyba při načítání konfigurace: %v", err)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("Chyba při připojování k MySQL: %v", err)
	}
	return db, nil
}

func (m *realMySQLRepo) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	logrus.Infof("Executing query: %s", query)
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("Error executing query: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("Error getting columns: %v", err)
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("Error scanning row: %v", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
		logrus.Infof("Retrieved row: %v", row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error with rows: %v", err)
	}

	return results, nil
}

func (m *realMySQLRepo) FetchData() ([]map[string]interface{}, error) {
	return nil, nil // Simulace prázdného výsledku
}

func (m *realMySQLRepo) Close() error {
	return nil
}

type realNeo4jRepo struct {
	driver neo4jDriver.Driver
}

func setupNeo4jConnection() (neo4jDriver.Driver, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("Chyba při načítání konfigurace: %v", err)
	}
	driver, err := neo4jDriver.NewDriver(cfg.Neo4j.URI, neo4jDriver.BasicAuth(cfg.Neo4j.User, cfg.Neo4j.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("Chyba při připojování k Neo4j: %v", err)
	}
	return driver, nil
}

func (m *realNeo4jRepo) StoreGraph(g *graph.GraphAggregate) error {
	// Start a new session for Neo4j
	session := m.driver.NewSession(neo4jDriver.SessionConfig{AccessMode: neo4jDriver.AccessModeWrite})
	defer session.Close()

	// Begin a transaction
	_, err := session.WriteTransaction(func(tx neo4jDriver.Transaction) (interface{}, error) {
		// Create nodes
		for _, node := range g.GetNodes() {
			nodeID := serialization.GenerateUniqueID() // Generování unikátního ID
			node.Properties["id"] = nodeID             // Nastavení unikátního ID v mapě vlastností
			_, err := tx.Run(
				"CREATE (n:Node {id: $id, type: $type, properties: $properties})",
				map[string]interface{}{
					"id":         nodeID,
					"type":       node.Type,
					"properties": node.Properties,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("Error creating node: %v", err)
			}
		}

		// Create relationships
		for _, rel := range g.GetRelationships() {
			_, err := tx.Run(
				"MATCH (a:Node {id: $fromId}), (b:Node {id: $toId}) CREATE (a)-[r:RELATION {type: $type, properties: $properties}]->(b)",
				map[string]interface{}{
					"fromId":     rel.SourceNode.ID,
					"toId":       rel.TargetNode.ID,
					"type":       rel.Type,
					"properties": rel.Properties,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("Error creating relationship: %v", err)
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("Error storing graph: %v", err)
	}

	return nil
}

func (m *realNeo4jRepo) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	return nil, nil
}

func (m *realNeo4jRepo) ExportGraph(query string) (interface{}, error) {
	return nil, nil
}

func (m *realNeo4jRepo) Close() error {
	return nil
}

func (m *realNeo4jRepo) FetchNodes(nodeType string) ([]map[string]interface{}, error) {
	// Simulace vrácení uzlů jako mapy
	return []map[string]interface{}{
		{"id": 1, "type": nodeType, "properties": map[string]interface{}{"name": "Node1"}},
		{"id": 2, "type": nodeType, "properties": map[string]interface{}{"name": "Node2"}},
	}, nil
}

const addr = "localhost:3000"

func TestIntegrationTransformRulesAndVisualization(t *testing.T) {
	ctx := context.Background()

	// Inicializace Neo4j klienta
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Chyba při načítání konfigurace: %v", err)
	}
	neo4jConfig := neo4j.Neo4jConfig{
		URI:      cfg.Neo4j.URI,
		User:     cfg.Neo4j.User,
		Password: cfg.Neo4j.Password,
	}
	neo4jClient, err := neo4j.NewNeo4jClient(neo4jConfig)
	if err != nil {
		t.Fatalf("Chyba při vytváření Neo4j klienta: %v", err)
	}
	defer neo4jClient.Close()

	// Spustit GraphQL server s neo4jPort
	go server.StartGraphQLServer(neo4jClient)

	mockRepo := &mockRuleRepo{}

	// Set up real MySQL connection
	db, err := setupMySQLConnection()
	if err != nil {
		t.Fatalf("Chyba při připojování k MySQL: %v", err)
	}
	defer db.Close()

	// Use the real MySQL connection
	mysqlRepo := &realMySQLRepo{db: db}

	// Use the Neo4j Client for storing the graph
	service := transformService.NewTransformService(mysqlRepo, neo4jClient, mockRepo)

	// Spustit transformaci
	err = service.TransformAndStore(ctx)
	assert.NoError(t, err)

	// Ověřit, že transformace proběhla
	results, err := neo4jClient.SearchNodes("")
	assert.NoError(t, err)
	assert.NotEmpty(t, results, "Nodes should be processed")

	// Spustit vizualizační server
	server := startVisualizationServer(neo4jClient)

	// Čekáme na uživatelský vstup před ukončením
	fmt.Printf("Test dokončen. Vizualizace je dostupná na http://localhost:3000\nStiskněte Ctrl+C pro ukončení...\n")

	// Čekáme na signál ukončení
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Ukončíme server
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("Chyba při ukončování serveru: %v", err)
	}
}

func startVisualizationServer(neo4jRepo ports.Neo4jPort) *http.Server {
	logrus.Infof("Začínám spouštět vizualizační server")
	mux := http.NewServeMux()

	// Aktualizace CORS nastavení
	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"}, // Povolí všechny origins pro testování
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Apollo-Require-Preflight"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(mux)

	// API endpoint pro data grafu
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		// Nastavení CORS hlaviček
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Apollo-Require-Preflight")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

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
				"id":         serialization.SerializeID(node.ID),
				"label":      node.Type,
				"properties": node.Properties,
			}
			response.Nodes = append(response.Nodes, nodeData)
			logrus.Infof("Přidávám uzel: %v", nodeData)
		}

		// Přidáme vztahy
		for _, rel := range g.GetRelationships() {
			relData := map[string]interface{}{
				"from":       serialization.SerializeID(rel.SourceNode.ID),
				"to":         serialization.SerializeID(rel.TargetNode.ID),
				"type":       rel.Type,
				"properties": rel.Properties,
			}
			response.Relationships = append(response.Relationships, relData)
			logrus.Infof("Přidávám vztah: %v", relData)
		}

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
		Handler: handler,
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

func GetConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"neo4j": map[string]string{
			"uri":      "bolt://localhost:7687",
			"username": "neo4j",
			"password": "password",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
