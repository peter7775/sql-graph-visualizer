package performance

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"sql-graph-visualizer/internal/application/ports"
)

func TestIndexInspectorDDL(t *testing.T) {
	for _, dialect := range []string{"mysql", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			inspector, err := NewIndexInspector(nil, dialect)
			if err != nil {
				t.Fatal(err)
			}
			spec := ports.IndexSpec{Name: "idx_orders_customer_id", Table: "orders", Columns: []string{"customer_id", "created_at"}}
			create, err := inspector.BuildCreateDDL(spec)
			if err != nil || create != "CREATE INDEX idx_orders_customer_id ON orders (customer_id, created_at)" {
				t.Fatalf("create=%q err=%v", create, err)
			}
			drop, err := inspector.BuildDropDDL(spec)
			want := "DROP INDEX idx_orders_customer_id ON orders"
			if dialect == "postgresql" {
				want = "DROP INDEX IF EXISTS idx_orders_customer_id"
			}
			if err != nil || drop != want {
				t.Fatalf("drop=%q err=%v", drop, err)
			}
			for _, bad := range []string{"", "a.b", "`orders`", `"orders"`, "a;DROP TABLE x", strings.Repeat("a", 64), "1bad"} {
				if _, err := inspector.BuildCreateDDL(ports.IndexSpec{Name: bad, Table: "orders", Columns: []string{"id"}}); err == nil {
					t.Errorf("accepted identifier %q", bad)
				}
				if _, err := inspector.IndexExists(context.Background(), bad, "idx"); err == nil {
					t.Errorf("accepted catalog identifier %q", bad)
				}
			}
			if _, err := inspector.BuildCreateDDL(ports.IndexSpec{Name: "idx", Table: "orders"}); err == nil {
				t.Error("accepted empty column list")
			}
		})
	}
	if _, err := NewIndexInspector(nil, "sqlite"); err == nil {
		t.Error("unsupported dialect accepted")
	}
}

func TestAnalyzerHeuristicIndexes(t *testing.T) {
	analyzer := &PerformanceAnalyzer{}
	suggestions := analyzer.generateIndexSuggestions(&ports.PerformanceAnalysis{
		Issues: []ports.PerformanceIssue{{
			Type:     ports.IssueTypeFullTableScan,
			Location: ports.BottleneckLocation{QueryPattern: "SELECT * FROM orders WHERE customer_id = ?"},
		}},
	})
	if len(suggestions) != 1 || suggestions[0].Confidence >= 0.5 ||
		suggestions[0].Implementation.Code != "CREATE INDEX idx_orders_customer_id ON orders (customer_id)" {
		t.Fatalf("unexpected heuristic recommendations %+v", suggestions)
	}
}

func TestCandidateColumns(t *testing.T) {
	tests := []struct {
		query string
		want  []indexCandidate
	}{
		{"SELECT * FROM orders WHERE customer_id = ?", []indexCandidate{{"orders", "customer_id", true}}},
		{"SELECT /*+ NO_INDEX(o) */ * FROM orders o JOIN customers AS c ON o.customer_id = c.id WHERE o.created_at >= $1", []indexCandidate{{"customers", "id", true}, {"orders", "customer_id", true}, {"orders", "created_at", false}}},
		{"UPDATE orders SET status = ? WHERE customer_id IN (?, ?) AND created_at BETWEEN ? AND ?", []indexCandidate{{"orders", "customer_id", true}, {"orders", "created_at", false}}},
		{"SELECT * FROM products WHERE name LIKE 'abc%'", []indexCandidate{{"products", "name", false}}},
		{"SELECT * FROM products WHERE name LIKE '%abc%'", []indexCandidate{}},
		{"SELECT * FROM orders WHERE customer_id = ? -- WHERE fake = ?\n", []indexCandidate{{"orders", "customer_id", true}}},
		{"SELECT * FROM orders o JOIN customers c ON customer_id = id", []indexCandidate{}},
		{"SELECT * FROM orders WHERE customer_id IN (SELECT id FROM customers)", nil},
		{"SELECT * FROM orders WHERE customer_id = ?; DROP TABLE orders", nil},
		{"SELECT * FROM public.orders WHERE customer_id = ?", nil},
		{"SELECT * FROM orders WHERE customer_id = ? OR status = ?", nil},
		{"SELECT * FROM `orders` WHERE customer_id = ?", nil},
		{"SELECT * FROM orders o, customers c WHERE customer_id = ?", nil},
		{"SELECT * FROM orders WHERE customer_id = 'unterminated", nil},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			if got := candidateColumns(test.query); !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %+v want %+v", got, test.want)
			}
		})
	}
}

func TestExplainLiterals(t *testing.T) {
	definition := ports.CustomQueryDefinition{
		Query:      "SELECT * FROM orders WHERE customer_id = $2 AND status = $1 AND note = '$1' /* $8 */",
		Parameters: []any{"O'Reilly", 12},
	}
	got, ok := explainLiterals(definition)
	if !ok || got != "SELECT * FROM orders WHERE customer_id = 12 AND status = 'O''Reilly' AND note = '$1' /* $8 */" {
		t.Fatalf("literal substitution %q %v", got, ok)
	}
	for _, parameter := range []any{struct{}{}, "a\\b", "a\x00b"} {
		if _, ok := explainLiterals(ports.CustomQueryDefinition{Query: "SELECT * FROM t WHERE c=$1", Parameters: []any{parameter}}); ok {
			t.Errorf("accepted unsafe literal %T", parameter)
		}
	}
	if _, ok := explainLiterals(ports.CustomQueryDefinition{Query: "SELECT * FROM t WHERE c=$2", Parameters: []any{1}}); ok {
		t.Error("accepted missing parameter")
	}
}

func TestMysqlScansTable(t *testing.T) {
	cases := []struct {
		access, extra string
		want          bool
	}{
		{"ALL", "Using where", true},
		{"index", "Using where; Backward index scan", true},
		{"index", "Using index", false},
		{"ref", "Using where", false},
		{"const", "", false},
	}
	for _, c := range cases {
		if got := mysqlScansTable(c.access, c.extra); got != c.want {
			t.Errorf("mysqlScansTable(%q, %q) = %v, want %v", c.access, c.extra, got, c.want)
		}
	}
}
