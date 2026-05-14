/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */

package factories

import (
	"testing"

	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/infrastructure/persistence/mssql"
	"sql-graph-visualizer/internal/infrastructure/persistence/oracle"
)

func TestCreateRepository_Oracle(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	repo, err := factory.CreateRepository(models.DatabaseTypeOracle)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	_, ok := repo.(*oracle.OracleDatabaseRepository)
	if !ok {
		t.Errorf("expected *oracle.OracleDatabaseRepository, got %T", repo)
	}
}

func TestCreateRepository_MySQL(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	repo, err := factory.CreateRepository(models.DatabaseTypeMySQL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestCreateRepository_PostgreSQL(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	repo, err := factory.CreateRepository(models.DatabaseTypePostgreSQL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestCreateRepository_Unsupported(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	_, err := factory.CreateRepository("sqlite")
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestCreateRepository_MSSQL(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	repo, err := factory.CreateRepository(models.DatabaseTypeMSSQL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	_, ok := repo.(*mssql.MSSQLDatabaseRepository)
	if !ok {
		t.Errorf("expected *mssql.MSSQLDatabaseRepository, got %T", repo)
	}
}

func TestGetSupportedDatabaseTypes(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	types := factory.GetSupportedDatabaseTypes()

	if len(types) != 4 {
		t.Fatalf("expected 4 supported types, got %d", len(types))
	}

	typeMap := make(map[models.DatabaseType]bool)
	for _, dt := range types {
		typeMap[dt] = true
	}

	if !typeMap[models.DatabaseTypeMySQL] {
		t.Error("MySQL should be in supported types")
	}
	if !typeMap[models.DatabaseTypePostgreSQL] {
		t.Error("PostgreSQL should be in supported types")
	}
	if !typeMap[models.DatabaseTypeOracle] {
		t.Error("Oracle should be in supported types")
	}
	if !typeMap[models.DatabaseTypeMSSQL] {
		t.Error("MSSQL should be in supported types")
	}
}

func TestNewDatabaseRepositoryFactory(t *testing.T) {
	factory := NewDatabaseRepositoryFactory()
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}
}
