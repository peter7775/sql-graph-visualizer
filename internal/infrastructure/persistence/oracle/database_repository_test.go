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

package oracle

import (
	"context"
	"strings"
	"testing"

	"sql-graph-visualizer/internal/domain/models"
)

// --- EscapeIdentifier tests ---

func TestEscapeIdentifier_Simple(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	result := repo.EscapeIdentifier("EMPLOYEES")
	expected := `"EMPLOYEES"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEscapeIdentifier_WithQuotes(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	// Double quotes inside should be escaped by doubling
	result := repo.EscapeIdentifier(`my"table`)
	expected := `"my""table"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEscapeIdentifier_Empty(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	result := repo.EscapeIdentifier("")
	expected := `""`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEscapeIdentifier_MultipleQuotes(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	result := repo.EscapeIdentifier(`a"b"c`)
	expected := `"a""b""c"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- GetQuoteChar tests ---

func TestGetQuoteChar(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	if repo.GetQuoteChar() != `"` {
		t.Errorf("expected double quote, got %s", repo.GetQuoteChar())
	}
}

// --- GetDatabaseType tests ---

func TestGetDatabaseType(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	if repo.GetDatabaseType() != models.DatabaseTypeOracle {
		t.Errorf("expected oracle, got %s", repo.GetDatabaseType())
	}
}

// --- TestConnection without active connection ---

func TestTestConnection_NoActiveConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	err := repo.TestConnection(context.Background())
	if err == nil {
		t.Fatal("expected error for no active connection")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Close without active connection ---

func TestClose_NoActiveConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	err := repo.Close()
	if err != nil {
		t.Errorf("Close on nil db should not error, got: %v", err)
	}
}

// --- Methods that require connection return error when db is nil ---

func TestGetTables_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetTables(context.Background(), models.DataFilteringConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetColumns_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetColumns(context.Background(), "test_table")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetForeignKeys_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetForeignKeys(context.Background(), "test_table")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetIndexes_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetIndexes(context.Background(), "test_table")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetConstraints_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetConstraints(context.Background(), "test_table")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDatabaseName_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetDatabaseName(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDatabaseVersion_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetDatabaseVersion(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSchemaNames_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetSchemaNames(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTableRowCount_NoConnection(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetTableRowCount(context.Background(), "test_table")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Not-implemented stubs ---

func TestSampleTableData_NotImplemented(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.SampleTableData(context.Background(), "test", 10)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAnalyzeColumnStatistics_NotImplemented(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.AnalyzeColumnStatistics(context.Background(), "test", "col")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTableSize_NotImplemented(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetTableSize(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetQueryExecutionPlan_NotImplemented(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.GetQueryExecutionPlan(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePermissions_NotImplemented(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	err := repo.ValidatePermissions(context.Background(), []string{"SELECT"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckUserPrivileges_NotImplemented(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	_, err := repo.CheckUserPrivileges(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- GetConnectionString tests ---

func TestGetConnectionString_ValidOracleConfig(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeOracle,
		Oracle: &models.OracleConfig{
			Host:        "dbhost",
			Port:        1521,
			ServiceName: "ORCL",
			Username:    "admin",
			Password:    "secret",
		},
	}

	connStr := repo.GetConnectionString(cfg)
	if !strings.Contains(connStr, "oracle://") {
		t.Errorf("expected oracle:// prefix, got %q", connStr)
	}
	if !strings.Contains(connStr, "dbhost:1521/ORCL") {
		t.Errorf("expected host:port/service in conn string, got %q", connStr)
	}
}

func TestGetConnectionString_WrongConfigType(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeMySQL,
		MySQL: &models.MySQLConfig{
			Host: "localhost",
		},
	}

	connStr := repo.GetConnectionString(cfg)
	if connStr != "" {
		t.Errorf("expected empty string for non-Oracle config, got %q", connStr)
	}
}

func TestGetConnectionString_NilOracleConfig(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeOracle,
	}

	connStr := repo.GetConnectionString(cfg)
	if connStr != "" {
		t.Errorf("expected empty string for nil Oracle config, got %q", connStr)
	}
}

// --- NewOracleDatabaseRepository tests ---

func TestNewOracleDatabaseRepository(t *testing.T) {
	repo := NewOracleDatabaseRepository()
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	oracleRepo, ok := repo.(*OracleDatabaseRepository)
	if !ok {
		t.Fatal("expected *OracleDatabaseRepository type")
	}
	if oracleRepo.db != nil {
		t.Error("expected nil db on new repository")
	}
}

// --- applyTableFiltering tests ---

func TestApplyTableFiltering_NoFilters(t *testing.T) {
	tables := []string{"EMPLOYEES", "DEPARTMENTS", "JOBS"}
	filters := models.DataFilteringConfig{}

	result := applyTableFiltering(tables, filters)
	if len(result) != 3 {
		t.Errorf("expected 3 tables, got %d", len(result))
	}
}

func TestApplyTableFiltering_Whitelist(t *testing.T) {
	tables := []string{"EMPLOYEES", "DEPARTMENTS", "JOBS", "LOCATIONS"}
	filters := models.DataFilteringConfig{
		TableWhitelist: []string{"EMPLOYEES", "JOBS"},
	}

	result := applyTableFiltering(tables, filters)
	if len(result) != 2 {
		t.Errorf("expected 2 tables, got %d", len(result))
	}
	for _, table := range result {
		if table != "EMPLOYEES" && table != "JOBS" {
			t.Errorf("unexpected table in result: %s", table)
		}
	}
}

func TestApplyTableFiltering_Blacklist(t *testing.T) {
	tables := []string{"EMPLOYEES", "DEPARTMENTS", "JOBS", "LOCATIONS"}
	filters := models.DataFilteringConfig{
		TableBlacklist: []string{"DEPARTMENTS", "LOCATIONS"},
	}

	result := applyTableFiltering(tables, filters)
	if len(result) != 2 {
		t.Errorf("expected 2 tables, got %d", len(result))
	}
	for _, table := range result {
		if table == "DEPARTMENTS" || table == "LOCATIONS" {
			t.Errorf("blacklisted table should not be in result: %s", table)
		}
	}
}

func TestApplyTableFiltering_BlacklistTakesPrecedence(t *testing.T) {
	tables := []string{"EMPLOYEES", "DEPARTMENTS", "JOBS"}
	filters := models.DataFilteringConfig{
		TableWhitelist: []string{"EMPLOYEES", "DEPARTMENTS"},
		TableBlacklist: []string{"DEPARTMENTS"},
	}

	result := applyTableFiltering(tables, filters)
	if len(result) != 1 {
		t.Errorf("expected 1 table, got %d", len(result))
	}
	if len(result) > 0 && result[0] != "EMPLOYEES" {
		t.Errorf("expected EMPLOYEES, got %s", result[0])
	}
}

func TestApplyTableFiltering_CaseInsensitive(t *testing.T) {
	tables := []string{"Employees", "departments"}
	filters := models.DataFilteringConfig{
		TableWhitelist: []string{"EMPLOYEES", "DEPARTMENTS"},
	}

	result := applyTableFiltering(tables, filters)
	if len(result) != 2 {
		t.Errorf("expected 2 tables (case insensitive), got %d", len(result))
	}
}

func TestApplyTableFiltering_EmptyTables(t *testing.T) {
	tables := []string{}
	filters := models.DataFilteringConfig{
		TableWhitelist: []string{"EMPLOYEES"},
	}

	result := applyTableFiltering(tables, filters)
	if len(result) != 0 {
		t.Errorf("expected 0 tables, got %d", len(result))
	}
}

// --- isInList tests ---

func TestIsInList_Found(t *testing.T) {
	if !isInList("EMPLOYEES", []string{"EMPLOYEES", "DEPARTMENTS"}) {
		t.Error("expected EMPLOYEES to be in list")
	}
}

func TestIsInList_NotFound(t *testing.T) {
	if isInList("JOBS", []string{"EMPLOYEES", "DEPARTMENTS"}) {
		t.Error("expected JOBS not to be in list")
	}
}

func TestIsInList_CaseInsensitive(t *testing.T) {
	if !isInList("employees", []string{"EMPLOYEES", "DEPARTMENTS"}) {
		t.Error("expected case-insensitive match")
	}
}

func TestIsInList_EmptyList(t *testing.T) {
	if isInList("EMPLOYEES", []string{}) {
		t.Error("expected false for empty list")
	}
}

func TestIsInList_EmptyItem(t *testing.T) {
	if isInList("", []string{"EMPLOYEES"}) {
		t.Error("expected false for empty item")
	}
}

// --- Connect with invalid config tests ---

func TestConnect_InvalidConfigType(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeMySQL,
		MySQL: &models.MySQLConfig{
			Host: "localhost",
		},
	}

	_, err := repo.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for non-Oracle config")
	}
	if !strings.Contains(err.Error(), "expected OracleConfig") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConnect_InvalidOracleConfig(t *testing.T) {
	repo := &OracleDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeOracle,
		Oracle: &models.OracleConfig{
			// Missing required fields
			Host: "localhost",
		},
	}

	_, err := repo.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid Oracle config")
	}
	if !strings.Contains(err.Error(), "invalid Oracle configuration") {
		t.Errorf("unexpected error: %v", err)
	}
}
