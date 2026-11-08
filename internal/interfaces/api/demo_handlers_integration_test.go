package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/models"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func TestDemoOptimizationDatabases(t *testing.T) {
	for _, dialect := range []string{"mysql", "postgresql"} {
		t.Run(dialect, func(t *testing.T) {
			env, driver := "MYSQL_TEST_DSN", "mysql"
			if dialect == "postgresql" {
				env, driver = "POSTGRES_TEST_DSN", "postgres"
			}
			if os.Getenv(env) == "" {
				t.Skip(env + " is not set")
			}
			db, err := sql.Open(driver, os.Getenv(env))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			if _, err := db.Exec("CREATE TABLE demo_index_fixture (id INTEGER PRIMARY KEY, customer_id INTEGER)"); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := db.Exec("DROP TABLE demo_index_fixture"); err != nil {
					t.Error(err)
				}
			}()
			inspector, err := performance.NewIndexInspector(db, dialect)
			if err != nil {
				t.Fatal(err)
			}
			demo := &models.LiveDemoConfig{Optimization: &models.LiveDemoOptimization{
				Name: "fixture", Indexes: []models.LiveDemoIndex{{Name: "idx_demo_fixture_customer", Table: "demo_index_fixture", Columns: []string{"customer_id"}}},
				// Invalid legacy SQL is ignored when neutral index specs are provided.
				Apply: []string{"DROP TABLE demo_index_fixture"}, Revert: []string{"DELETE FROM demo_index_fixture"},
			}}
			handler := NewDemoHandlers(logrus.New(), nil, db, demo, inspector)
			router := mux.NewRouter()
			handler.RegisterRoutes(router)
			state := handler.OptimizationState(context.Background())
			if !state.Available || !state.CanApply || state.Applied {
				t.Fatalf("initial state %+v", state)
			}
			for _, action := range []string{"apply", "apply", "revert", "revert"} {
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/demo/optimization/"+action, nil))
				if recorder.Code != http.StatusOK {
					t.Fatalf("%s status=%d body=%s", action, recorder.Code, recorder.Body.String())
				}
				var response struct {
					Data demoOptimizationInfo `json:"data"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Data.DatabaseType != dialect || response.Data.Applied != (action == "apply") {
					t.Fatalf("unexpected state %+v", response.Data)
				}
				if handler.OptimizationState(context.Background()).Applied != (action == "apply") {
					t.Fatal("exported state differs from HTTP response")
				}
			}
			legacy := &models.LiveDemoConfig{Optimization: &models.LiveDemoOptimization{
				Name:   "legacy",
				Apply:  []string{"CREATE INDEX idx_demo_fixture_customer ON demo_index_fixture (customer_id)"},
				Revert: []string{"DROP INDEX idx_demo_fixture_customer ON demo_index_fixture"},
			}}
			legacyHandler := NewDemoHandlers(logrus.New(), nil, db, legacy, inspector)
			for _, apply := range []bool{true, false} {
				recorder := httptest.NewRecorder()
				legacyHandler.changeOptimization(recorder, httptest.NewRequest(http.MethodPost, "/", nil), apply)
				if recorder.Code != http.StatusOK {
					t.Fatalf("legacy apply=%v failed: %s", apply, recorder.Body.String())
				}
			}
		})
	}
}
