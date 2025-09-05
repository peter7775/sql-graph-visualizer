/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

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

func (m *mockRuleRepo) GetAllRules(ctx context.Context) ([]*transformAggregates.RuleAggregate, error) {
	return []*transformAggregates.RuleAggregate{
		{
			Rule: transformObjects.TransformRule{
				Name:       "php_actions_to_nodes",
				RuleType:   transformObjects.NodeRule,
				SourceSQL:  "SELECT DISTINCT au.id as id, au.id_typu, au.infix, au.nazev, au.prefix FROM testdata_uzly au WHERE au.id_typu = 17",
				TargetType: "NodePHPAction",
				FieldMappings: map[string]string{
					"id":      "id",
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
				SourceSQL:  "SELECT DISTINCT au.id_node as id, au.php_code FROM testdata_uzly_php_action au JOIN testdata_uzly aupa ON au.id_node = aupa.id",
				TargetType: "PHPAction",
				FieldMappings: map[string]string{
					"id":       "id",
					"php_code": "php_code",
				},
			},
		},
		{
			Rule: transformObjects.TransformRule{
				Name:          "php_action_relationship",
				RuleType:      transformObjects.RelationshipRule,
				RelationType:  "AKCE",
				Direction:     transformObjects.Outgoing,
				SourceNode:    &transformObjects.NodeMapping{Type: "PHPAction", Key: "id", TargetField: "id"},
				TargetNode:    &transformObjects.NodeMapping{Type: "NodePHPAction", Key: "id", TargetField: "id"},
				FieldMappings: map[string]string{"id": "id"},
			},
		},
	}, nil
}

func (m *mockRuleRepo) SaveRule(ctx context.Context, rule *transformAggregates.RuleAggregate) error {
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
		return nil, fmt.Errorf("Error loading configuration: %v", err)
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
		return nil, fmt.Errorf("Error connecting to MySQL: %v", err)
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
	return nil, nil // Simulate empty result
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
		return nil, fmt.Errorf("Error loading configuration: %v", err)
	}
	driver, err := neo4jDriver.NewDriver(cfg.Neo4j.URI, neo4jDriver.BasicAuth(cfg.Neo4j.User, cfg.Neo4j.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Neo4j: %v", err)
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

	// Test připojení
	err = db.Ping()
	if err != nil {
		t.Fatalf("Nelze pingout MySQL: %v", err)
	}

	// Zkusit jednoduchý dotaz
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		t.Fatalf("Nelze vykonat SHOW TABLES: %v", err)
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("Chyba při čtení tabulek: %v", err)
		}
		tables = append(tables, table)
	}
	t.Logf("Nalezené tabulky: %v", tables)

	// Use the real MySQL connection
	mysqlRepo := &realMySQLRepo{db: db}

	// Kontrola SQL dotazů
	results1, err := mysqlRepo.ExecuteQuery("SELECT DISTINCT au.id as id, au.id_typu, au.infix, au.nazev, au.prefix FROM testdata_uzly au WHERE au.id_typu = 17")
	assert.NoError(t, err)
	t.Logf("První SQL dotaz vrátil %d záznamů: %v", len(results1), results1)

	// Test druhého SQL dotazu
	results2, err := mysqlRepo.ExecuteQuery("SELECT DISTINCT au.id_node as id, au.php_code FROM testdata_uzly_php_action au JOIN testdata_uzly aupa ON au.id_node = aupa.id")
	assert.NoError(t, err)
	t.Logf("Druhý SQL dotaz vrátil %d záznamů: %v", len(results2), results2)

	// Kontrola vytvoření uzlů před vytvořením relací
	session := neo4jClient.GetDriver().NewSession(neo4jDriver.SessionConfig{})
	defer session.Close()

	// Před vytvořením relací vyčistíme existující data
	cleanupResult, err := session.Run(`
		MATCH (n)
		DETACH DELETE n
	`, map[string]interface{}{})
	assert.NoError(t, err)
	cleanupResult.Consume()

	// Vytvoření uzlů NodePHPAction
	createNodesResult1, err := session.Run(`
		UNWIND $nodes as node
		CREATE (n:NodePHPAction)
		SET n = node
		RETURN count(n)
	`, map[string]interface{}{
		"nodes": results1,
	})
	assert.NoError(t, err)
	if createNodesResult1.Next() {
		count := createNodesResult1.Record().GetByIndex(0).(int64)
		t.Logf("Created %d NodePHPAction nodes directly", count)
	}

	// Vytvoření uzlů PHPAction
	createNodesResult2, err := session.Run(`
		UNWIND $nodes as node
		CREATE (n:PHPAction)
		SET n = node
		RETURN count(n)
	`, map[string]interface{}{
		"nodes": results2,
	})
	assert.NoError(t, err)
	if createNodesResult2.Next() {
		count := createNodesResult2.Record().GetByIndex(0).(int64)
		t.Logf("Created %d PHPAction nodes directly", count)
	}

	// Kontrola počtu vytvořených uzlů PHPAction
	countPHPActionResult, err := session.Run(`
		MATCH (n:PHPAction)
		RETURN count(n) as count, collect(n.id) as ids
	`, map[string]interface{}{})
	assert.NoError(t, err)
	if countPHPActionResult.Next() {
		count := countPHPActionResult.Record().GetByIndex(0).(int64)
		ids := countPHPActionResult.Record().GetByIndex(1)
		t.Logf("Found %d PHPAction nodes: %v", count, ids)
	}

	// Kontrola počtu vytvořených uzlů NodePHPAction
	countNodePHPActionResult, err := session.Run(`
		MATCH (n:NodePHPAction)
		RETURN count(n) as count, collect(n.id) as ids
	`, map[string]interface{}{})
	assert.NoError(t, err)
	if countNodePHPActionResult.Next() {
		count := countNodePHPActionResult.Record().GetByIndex(0).(int64)
		ids := countNodePHPActionResult.Record().GetByIndex(1)
		t.Logf("Found %d NodePHPAction nodes: %v", count, ids)
	}

	// Kontrola párů uzlů se stejnými ID
	matchingNodesResult, err := session.Run(`
		MATCH (source:PHPAction), (target:NodePHPAction)
		WHERE source.id = target.id
		RETURN count(*) as count, collect({source: source.id, target: target.id}) as matches
	`, map[string]interface{}{})
	assert.NoError(t, err)
	if matchingNodesResult.Next() {
		count := matchingNodesResult.Record().GetByIndex(0).(int64)
		matches := matchingNodesResult.Record().GetByIndex(1)
		t.Logf("Found %d matching node pairs: %v", count, matches)
	}

	// Vytvoření relací
	createRelResult, err := session.Run(`
		MATCH (source:PHPAction), (target:NodePHPAction)
		WHERE source.id = target.id
		CREATE (source)-[r:AKCE]->(target)
		RETURN count(r) as count
	`, map[string]interface{}{})
	assert.NoError(t, err)
	if createRelResult.Next() {
		count := createRelResult.Record().GetByIndex(0).(int64)
		t.Logf("Created %d relationships directly", count)
	}

	// Kontrola vytvořených relací
	checkRelResult, err := session.Run(`
		MATCH (source:PHPAction)-[r:AKCE]->(target:NodePHPAction)
		RETURN source.id, target.id, type(r)
	`, map[string]interface{}{})
	assert.NoError(t, err)

	for checkRelResult.Next() {
		record := checkRelResult.Record()
		t.Logf("Relationship: %v -> %v (type: %v)",
			record.GetByIndex(0),
			record.GetByIndex(1),
			record.GetByIndex(2))
	}

	// Use the Neo4j Client for storing the graph
	service := transformService.NewTransformService(mysqlRepo, neo4jClient, mockRepo)

	// Spustit transformaci
	err = service.TransformAndStore(ctx)
	assert.NoError(t, err)

	// Spustit vizualizační server
	server := startVisualizationServer(t)

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

func startVisualizationServer(t *testing.T) *http.Server {
	addr := "localhost:3000"
	mux := http.NewServeMux()

	// Nastavení CORS
	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(mux)

	// Přidáme endpoint pro konfiguraci
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.Load()
		if err != nil {
			http.Error(w, "Chyba při načítání konfigurace", http.StatusInternalServerError)
			return
		}

		config := map[string]interface{}{
			"neo4j": map[string]string{
				"uri":      cfg.Neo4j.URI,
				"username": cfg.Neo4j.User,
				"password": cfg.Neo4j.Password,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Statické soubory
	webRoot := filepath.Join(findProjectRoot(), "internal", "interfaces", "web")
	logrus.Infof("Používám web root: %s", webRoot)

	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Hlavní stránka
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		logrus.Infof("Požadavek na hlavní stránku")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	// Vytvoříme listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Warnf("Port %s je obsazený: %v", addr, err)
		exec.Command("fuser", "-k", "3000/tcp").Run()
		time.Sleep(time.Second)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			logrus.Fatalf("Nelze vytvořit listener: %v", err)
		}
	}
	logrus.Infof("Listener vytvořen na %s", addr)

	server := &http.Server{
		Handler: handler,
	}

	go func() {
		logrus.Warnf("Spouštím server na %s", addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server ukončen s chybou: %v", err)
		}
	}()

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
