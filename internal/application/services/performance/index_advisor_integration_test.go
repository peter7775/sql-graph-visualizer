package performance

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

func TestIndexAdvisorDatabases(t *testing.T) {
	for _, dialect := range []string{"mysql", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			env, driver := "MYSQL_TEST_DSN", "mysql"
			if dialect == "postgresql" {
				env, driver = "POSTGRES_TEST_DSN", "postgres"
			}
			dsn := os.Getenv(env)
			if dsn == "" {
				t.Skip(env + " is not set; requires an empty disposable database")
			}
			db, err := sql.Open(driver, dsn)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			exec := func(statement string, args ...any) {
				t.Helper()
				if _, err := db.ExecContext(ctx, statement, args...); err != nil {
					t.Fatalf("fixture SQL failed: %v", err)
				}
			}
			// CREATE deliberately fails when these names already exist, preventing
			// accidental modification of an existing application dataset.
			for _, table := range []string{"categories", "customers", "products", "orders", "order_items"} {
				columns := "id INTEGER PRIMARY KEY"
				switch table {
				case "categories":
					columns += ", name VARCHAR(120) DEFAULT 'category'"
				case "customers":
					columns += ", email VARCHAR(120), first_name VARCHAR(50) DEFAULT 'test', country VARCHAR(2) DEFAULT 'CZ', city VARCHAR(60) DEFAULT 'Prague', last_login_at TIMESTAMP"
				case "products":
					columns += ", category_id INTEGER DEFAULT 7, name VARCHAR(120) DEFAULT 'product', price INTEGER DEFAULT 10, stock_quantity INTEGER DEFAULT 500"
				case "orders":
					columns += ", customer_id INTEGER, created_at TIMESTAMP DEFAULT '2024-03-04 00:00:00', status VARCHAR(16) DEFAULT 'paid', total_amount INTEGER DEFAULT 100"
				case "order_items":
					columns += ", order_id INTEGER, product_id INTEGER, quantity INTEGER DEFAULT 1, unit_price INTEGER DEFAULT 10"
				}
				exec("CREATE TABLE " + table + " (" + columns + ")")
				tableName := table
				t.Cleanup(func() {
					if _, err := db.Exec("DROP TABLE " + tableName); err != nil {
						t.Errorf("fixture cleanup: %v", err)
					}
				})
			}
			for _, table := range []string{"customers", "products", "orders", "order_items"} {
				for batch := 0; batch < 30; batch++ {
					values := make([]string, 0, 1000)
					for row := 1; row <= 1000; row++ {
						id := batch*1000 + row
						value := fmt.Sprintf("(%d)", id)
						switch table {
						case "orders":
							value = fmt.Sprintf("(%d,%d)", id, id%10000)
						case "order_items":
							value = fmt.Sprintf("(%d,%d,%d)", id, id%1000, id%700)
						}
						values = append(values, value)
					}
					columns := "(id)"
					switch table {
					case "orders":
						columns = "(id,customer_id)"
					case "order_items":
						columns = "(id,order_id,product_id)"
					}
					exec("INSERT INTO " + table + columns + " VALUES " + strings.Join(values, ","))
				}
			}
			exec("CREATE INDEX idx_orders_created_at ON orders (created_at)")
			exec("CREATE INDEX idx_order_items_order ON order_items (order_id)")
			exec("CREATE INDEX idx_products_category ON products (category_id)")
			exec("CREATE INDEX idx_customers_country_city ON customers (country, city)")
			exec("INSERT INTO categories (id) VALUES (7)")
			for _, table := range []string{"orders", "order_items"} {
				if dialect == "postgresql" {
					exec("ANALYZE " + table)
				} else {
					exec("ANALYZE TABLE " + table)
				}
			}
			inspector, err := NewIndexInspector(db, dialect)
			if err != nil {
				t.Fatal(err)
			}
			placeholder := "?"
			if dialect == "postgresql" {
				placeholder = "$1"
			}
			definitions := []ports.CustomQueryDefinition{
				{Query: "SELECT * FROM orders WHERE customer_id = " + placeholder, Parameters: []any{42}},
				{Query: "SELECT * FROM order_items WHERE product_id = " + placeholder, Parameters: []any{42}},
			}
			result := &ports.BenchmarkResult{}
			for _, definition := range definitions {
				result.QueryResults = append(result.QueryResults, ports.QueryPerformance{
					QueryPattern: definition.Query, QueryType: "SELECT", AverageTime: 40 * time.Millisecond, ExecutionCount: 100,
				})
			}
			advisor := NewIndexAdvisor(newTestLogger(), db, inspector)
			compositeQuery := "SELECT * FROM orders WHERE total_amount >= " + placeholder + " AND customer_id = " + placeholder + " AND status = " + placeholder
			if dialect == "postgresql" {
				compositeQuery = "SELECT * FROM orders WHERE total_amount >= $1 AND customer_id = $2 AND status = $3"
			}
			compositeDefinitions := []ports.CustomQueryDefinition{{Query: compositeQuery, Parameters: []any{10, 42, "paid"}}}
			compositeResult := &ports.BenchmarkResult{QueryResults: []ports.QueryPerformance{{
				QueryPattern: compositeQuery, QueryType: "SELECT", AverageTime: 40 * time.Millisecond, ExecutionCount: 100,
			}}}
			composite, err := advisor.SuggestIndexes(ctx, compositeResult, compositeDefinitions)
			if err != nil || len(composite) != 1 || strings.Join(composite[0].Columns, ",") != "customer_id,status,total_amount" {
				t.Fatalf("unexpected composite recommendation: %+v err=%v", composite, err)
			}
			fallback, err := advisor.SuggestIndexes(ctx, result, []ports.CustomQueryDefinition{
				{Query: definitions[0].Query}, {Query: definitions[1].Query},
			})
			if err != nil || len(fallback) != 2 || fallback[0].Source != "heuristic" || fallback[0].Confidence >= 0.9 {
				t.Fatalf("expected parameter-error fallback: %+v err=%v", fallback, err)
			}
			suggestions, err := advisor.SuggestIndexes(ctx, result, definitions)
			if err != nil || len(suggestions) != 2 {
				t.Fatalf("suggestions=%+v err=%v", suggestions, err)
			}
			for _, suggestion := range suggestions {
				t.Logf("%s: %s source=%s confidence=%.2f", dialect, suggestion.Statements[0], suggestion.Source, suggestion.Confidence)
				if suggestion.Source != "explain" || suggestion.Applied {
					t.Errorf("expected confirmed unapplied suggestion %+v", suggestion)
				}
				exec(suggestion.Statements[0])
				exists, err := inspector.IndexExists(ctx, suggestion.Table, suggestion.ID)
				if err != nil || !exists {
					t.Fatalf("applied index missing: exists=%v err=%v", exists, err)
				}
			}
			after, err := advisor.SuggestIndexes(ctx, result, definitions)
			if err != nil || after == nil || len(after) != 0 {
				t.Fatalf("existing indexes must not be recommended: %+v err=%v", after, err)
			}
			for _, suggestion := range suggestions {
				exec(suggestion.Rollback[0])
				exists, err := inspector.IndexExists(ctx, suggestion.Table, suggestion.ID)
				if err != nil || exists {
					t.Fatalf("dropped index still present: exists=%v err=%v", exists, err)
				}
			}
			testConfiguredDemoAdvice(ctx, t, advisor, dialect)
		})
	}
}

func testConfiguredDemoAdvice(ctx context.Context, t *testing.T, advisor *IndexAdvisor, dialect string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "config", "demo-config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config models.Config
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	for _, set := range config.Performance.Benchmarks.CustomQueries {
		definitions := []ports.CustomQueryDefinition{}
		result := &ports.BenchmarkResult{}
		for _, query := range set.Queries {
			sqlText := query.Query
			if dialect == "postgresql" {
				for position := 1; strings.Contains(sqlText, "?"); position++ {
					sqlText = strings.Replace(sqlText, "?", fmt.Sprintf("$%d", position), 1)
				}
			}
			definitions = append(definitions, ports.CustomQueryDefinition{Query: sqlText, Parameters: query.Parameters})
			result.QueryResults = append(result.QueryResults, ports.QueryPerformance{
				QueryPattern: sqlText, QueryType: strings.ToUpper(strings.Fields(sqlText)[0]), AverageTime: 40 * time.Millisecond, ExecutionCount: int64(query.Weight),
			})
		}
		suggestions, err := advisor.SuggestIndexes(ctx, result, definitions)
		if err != nil {
			t.Fatal(err)
		}
		for _, suggestion := range suggestions {
			t.Logf("demo %s/%s: %s (%s)", dialect, set.Name, suggestion.Statements[0], suggestion.Source)
			if suggestion.Table != "orders" && suggestion.Table != "order_items" {
				t.Errorf("unexpected demo recommendation: %+v", suggestion)
			}
		}
	}
}
