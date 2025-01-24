package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/peter7775/alevisualizer/internal/application/services/transform"
	"github.com/peter7775/alevisualizer/internal/domain/aggregates/graph"
	transformagg "github.com/peter7775/alevisualizer/internal/domain/aggregates/transform"
	"github.com/peter7775/alevisualizer/internal/domain/repositories/mysql"
	transformvo "github.com/peter7775/alevisualizer/internal/domain/valueobjects/transform"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type mockRuleRepo struct{}

func (m *mockRuleRepo) GetAllRules(ctx context.Context) ([]*transformagg.TransformRuleAggregate, error) {
	rules := []*transformagg.TransformRuleAggregate{
		// Pravidlo pro transformaci uživatelů
		{
			Rule: transformvo.TransformRule{
				Name:        "users_to_persons",
				RuleType:    transformvo.NodeRule,
				SourceTable: "users",
				TargetType:  "Person",
				FieldMappings: map[string]string{
					"id":    "id",
					"name":  "name",
					"email": "email",
					"role":  "role",
				},
			},
		},
		// Pravidlo pro transformaci oddělení
		{
			Rule: transformvo.TransformRule{
				Name:        "departments_to_departments",
				RuleType:    transformvo.NodeRule,
				SourceTable: "departments",
				TargetType:  "Department",
				FieldMappings: map[string]string{
					"id":   "id",
					"name": "name",
					"code": "code",
				},
			},
		},
		// Pravidlo pro transformaci vztahů uživatel-oddělení
		{
			Rule: transformvo.TransformRule{
				Name:        "user_departments_to_works_in",
				RuleType:    transformvo.RelationshipRule,
				SourceTable: "user_departments",
				TargetType:  "WORKS_IN",
				Direction:   transformvo.Outgoing,
				SourceNode: &transformvo.NodeMapping{
					Type:        "Person",
					Key:         "user_id",
					TargetField: "id",
				},
				TargetNode: &transformvo.NodeMapping{
					Type:        "Department",
					Key:         "department_id",
					TargetField: "id",
				},
				FieldMappings: map[string]string{
					"role":       "role",
					"start_date": "start_date",
				},
			},
		},
		// Pravidlo pro transformaci vztahů mezi odděleními
		{
			Rule: transformvo.TransformRule{
				Name:        "department_collaborations_to_collaborates_with",
				RuleType:    transformvo.RelationshipRule,
				SourceTable: "department_collaborations",
				TargetType:  "COLLABORATES_WITH",
				Direction:   transformvo.Outgoing,
				SourceNode: &transformvo.NodeMapping{
					Type:        "Department",
					Key:         "department_id_1",
					TargetField: "id",
				},
				TargetNode: &transformvo.NodeMapping{
					Type:        "Department",
					Key:         "department_id_2",
					TargetField: "id",
				},
				FieldMappings: map[string]string{
					"start_date":   "start_date",
					"num_projects": "num_projects",
				},
			},
		},
	}
	return rules, nil
}

func (m *mockRuleRepo) SaveRule(ctx context.Context, rule *transformagg.TransformRuleAggregate) error {
	return nil
}

func (m *mockRuleRepo) DeleteRule(ctx context.Context, ruleID string) error {
	return nil
}

func (m *mockRuleRepo) UpdateRulePriority(ctx context.Context, ruleID string, priority int) error {
	return nil
}

type mockMySQLRepo struct {
	db *sql.DB
}

func NewMockMySQLRepo(db *sql.DB) *mockMySQLRepo {
	return &mockMySQLRepo{db: db}
}

func (m *mockMySQLRepo) FetchData() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Načtení uživatelů
	rows, err := m.db.Query("SELECT * FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		data := make(map[string]interface{})
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, col := range cols {
			if values[i] != nil {
				data[col] = values[i]
			}
		}
		// Přidáme informaci o zdrojové tabulce
		data["_table"] = "users"
		results = append(results, data)
	}

	// Načtení oddělení
	rows, err = m.db.Query("SELECT * FROM departments")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err = rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		data := make(map[string]interface{})
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, col := range cols {
			if values[i] != nil {
				data[col] = values[i]
			}
		}
		// Přidáme informaci o zdrojové tabulce
		data["_table"] = "departments"
		results = append(results, data)
	}

	// Načtení vztahů uživatel-oddělení
	rows, err = m.db.Query("SELECT * FROM user_departments")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err = rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		data := make(map[string]interface{})
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, col := range cols {
			if values[i] != nil {
				data[col] = values[i]
			}
		}
		// Přidáme informaci o zdrojové tabulce
		data["_table"] = "user_departments"
		results = append(results, data)
	}

	// Načtení vztahů mezi odděleními
	rows, err = m.db.Query("SELECT * FROM department_collaborations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err = rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		data := make(map[string]interface{})
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, col := range cols {
			if values[i] != nil {
				data[col] = values[i]
			}
		}
		// Přidáme informaci o zdrojové tabulce
		data["_table"] = "department_collaborations"
		results = append(results, data)
	}

	log.Printf("Načteno %d záznamů z MySQL", len(results))
	return results, nil
}

func (m *mockMySQLRepo) Close() error {
	return m.db.Close()
}

type mockNeo4jRepo struct {
	nodes []*graph.GraphAggregate
}

func NewMockNeo4jRepo() *mockNeo4jRepo {
	return &mockNeo4jRepo{
		nodes: make([]*graph.GraphAggregate, 0),
	}
}

func (m *mockNeo4jRepo) StoreGraph(g *graph.GraphAggregate) error {
	log.Printf("Ukládám graf s %d uzly a %d vztahy", len(g.GetNodes()), len(g.GetRelationships()))
	m.nodes = []*graph.GraphAggregate{g}
	return nil
}

func (m *mockNeo4jRepo) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	log.Printf("Hledám uzly, k dispozici %d grafů", len(m.nodes))
	if len(m.nodes) == 0 {
		return nil, fmt.Errorf("žádná data nejsou k dispozici")
	}

	// Logování detailů o uzlech
	for _, g := range m.nodes {
		nodes := g.GetNodes()
		relationships := g.GetRelationships()
		log.Printf("Graf obsahuje %d uzlů a %d vztahů", len(nodes), len(relationships))

		for _, node := range nodes {
			log.Printf("Uzel: Type=%s, ID=%v, Properties=%v", node.Type, node.ID, node.Properties)
		}

		for _, rel := range relationships {
			log.Printf("Vztah: Type=%s, From=%v, To=%v, Properties=%v",
				rel.Type, rel.SourceNode.ID, rel.TargetNode.ID, rel.Properties)
		}
	}

	return m.nodes, nil
}

func (m *mockNeo4jRepo) ExportGraph(query string) (interface{}, error) {
	if len(m.nodes) == 0 {
		return nil, fmt.Errorf("žádná data nejsou k dispozici")
	}
	return m.nodes[0], nil
}

func (m *mockNeo4jRepo) Close() error {
	return nil
}

func getMySQLConfig() *mysql.MySQLConfig {
	// Načtení konfigurace ze souboru
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Nelze získat pracovní adresář: %v", err)
	}

	// Hledání config.yml
	var configPath string
	for {
		testPath := filepath.Join(wd, "config.yml")
		if _, err := os.Stat(testPath); err == nil {
			configPath = testPath
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			log.Fatalf("Nelze najít config.yml")
			return nil
		}
		wd = parent
	}

	// Načtení konfigurace
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Nelze načíst konfigurační soubor: %v", err)
		return nil
	}

	// Parsování YAML
	var config struct {
		MySQL struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			Database string `yaml:"database"`
		} `yaml:"mysql"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Nelze parsovat konfigurační soubor: %v", err)
		return nil
	}

	return &mysql.MySQLConfig{
		Host:     config.MySQL.Host,
		Port:     config.MySQL.Port,
		User:     config.MySQL.User,
		Password: config.MySQL.Password,
		Database: config.MySQL.Database,
	}
}

func cleanDatabase(t *testing.T, db *sql.DB) {
	tables := []string{
		"department_collaborations",
		"user_departments",
		"users",
		"departments",
	}

	for _, table := range tables {
		_, err := db.Exec("DROP TABLE IF EXISTS " + table)
		assert.NoError(t, err)
	}
}

func initTestDatabase(t *testing.T, db *sql.DB) {
	// Vyčištění databáze
	cleanDatabase(t, db)

	// Načtení init.sql z kořene projektu
	wd, err := os.Getwd()
	assert.NoError(t, err)

	// Hledáme cestu k testdata/mysql/init.sql
	var initSQLPath string
	for {
		testPath := filepath.Join(wd, "testdata", "mysql", "init.sql")
		if _, err := os.Stat(testPath); err == nil {
			initSQLPath = testPath
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("Nelze najít testdata/mysql/init.sql")
			return
		}
		wd = parent
	}

	// Načtení init.sql
	initSQL, err := os.ReadFile(initSQLPath)
	assert.NoError(t, err)

	// Rozdělení na jednotlivé příkazy a spuštění
	statements := strings.Split(string(initSQL), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err = db.Exec(stmt)
		assert.NoError(t, err)
	}
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

func startVisualizationServer(t *testing.T, neo4jRepo *mockNeo4jRepo) {
	mux := http.NewServeMux()

	// API endpoint pro data grafu
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Požadavek na API endpoint /api/graph")

		// Získáme celý graf
		graphInterface, err := neo4jRepo.ExportGraph("")
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
		}

		// Nastavíme hlavičky
		w.Header().Set("Content-Type", "application/json")

		// Vypíšeme data pro debug
		log.Printf("Odesílám odpověď: %d uzlů, %d vztahů", len(response.Nodes), len(response.Relationships))

		// Odešleme odpověď
		json.NewEncoder(w).Encode(response)
	})

	// Servírujeme statické soubory
	webRoot := filepath.Join(findProjectRoot(), "internal", "interfaces", "web")
	log.Printf("Používám web root: %s", webRoot)

	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Servírujeme HTML stránku
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	// Spustíme server na pozadí
	addr := "localhost:8081"
	log.Printf("Spouštím server na %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("Server ukončen s chybou: %v", err)
		}
	}()

	// Počkáme chvíli na nastartování serveru
	time.Sleep(100 * time.Millisecond)
	log.Printf("Vizualizace je dostupná na http://%s", addr)
}

func TestTransformService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Inicializace MySQL s testovacími daty
	config := getMySQLConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	log.Printf("Připojuji se k MySQL na %s:%d", config.Host, config.Port)
	db, err := sql.Open("mysql", dsn)
	assert.NoError(t, err)
	defer db.Close()

	// Test připojení
	err = db.Ping()
	if err != nil {
		t.Fatalf("Nelze se připojit k MySQL: %v", err)
	}
	log.Printf("Připojení k MySQL úspěšné")

	// Inicializace testovací databáze
	log.Printf("Inicializuji testovací databázi")
	initTestDatabase(t, db)

	// Kontrola dat v MySQL
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	assert.NoError(t, err)
	log.Printf("Počet uživatelů v MySQL: %d", count)

	err = db.QueryRow("SELECT COUNT(*) FROM departments").Scan(&count)
	assert.NoError(t, err)
	log.Printf("Počet oddělení v MySQL: %d", count)

	mysqlRepo := NewMockMySQLRepo(db)
	neo4jRepo := NewMockNeo4jRepo()

	// Vytvoření služby
	service := transform.NewTransformService(mysqlRepo, neo4jRepo, &mockRuleRepo{})

	// Test transformace a uložení
	log.Printf("Spouštím transformaci")
	err = service.TransformAndStore(context.Background())
	assert.NoError(t, err)

	// Ověření výsledků v Neo4j
	log.Printf("Kontroluji výsledky v Neo4j")
	results, err := neo4jRepo.SearchNodes("")
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	// Kontrola počtu uzlů podle typu
	nodes := results[0].GetNodes()
	log.Printf("Celkový počet uzlů v Neo4j: %d", len(nodes))

	personNodes := 0
	deptNodes := 0
	for _, node := range nodes {
		switch node.Type {
		case "Person":
			personNodes++
			log.Printf("Nalezen Person uzel: %v", node.Properties)
		case "Department":
			deptNodes++
			log.Printf("Nalezen Department uzel: %v", node.Properties)
		}
	}
	assert.Equal(t, 2, personNodes, "Očekávány 2 uzly typu Person")
	assert.Equal(t, 2, deptNodes, "Očekávány 2 uzly typu Department")

	// Spuštění vizualizačního serveru
	startVisualizationServer(t, neo4jRepo)

	// Čekáme na uživatelský vstup před ukončením
	fmt.Printf("Test dokončen. Vizualizace je dostupná na http://%s\n", addr)
	fmt.Println("Stiskněte Enter pro ukončení...")
	fmt.Scanln()
}
