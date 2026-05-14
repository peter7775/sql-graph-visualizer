/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

package mssql

import (
	"context"
	"strings"
	"testing"

	"sql-graph-visualizer/internal/domain/models"
)

// --- EscapeIdentifier tests ---

func TestEscapeIdentifier_Simple(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	result := repo.EscapeIdentifier("Users")
	expected := "[Users]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEscapeIdentifier_WithBrackets(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	result := repo.EscapeIdentifier("my]table")
	expected := "[my]]table]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEscapeIdentifier_Empty(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	result := repo.EscapeIdentifier("")
	expected := "[]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- GetQuoteChar tests ---

func TestGetQuoteChar(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	if repo.GetQuoteChar() != "[" {
		t.Errorf("expected [, got %s", repo.GetQuoteChar())
	}
}

// --- GetDatabaseType tests ---

func TestGetDatabaseType(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	if repo.GetDatabaseType() != models.DatabaseTypeMSSQL {
		t.Errorf("expected mssql, got %s", repo.GetDatabaseType())
	}
}

// --- TestConnection without active connection ---

func TestTestConnection_NoActiveConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
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
	repo := &MSSQLDatabaseRepository{}
	err := repo.Close()
	if err != nil {
		t.Errorf("Close on nil db should not error, got: %v", err)
	}
}

// --- Methods that require connection return error when db is nil ---

func TestGetTables_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetTables(context.Background(), models.DataFilteringConfig{})
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected 'no active database connection' error, got: %v", err)
	}
}

func TestGetColumns_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetColumns(context.Background(), "test_table")
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetForeignKeys_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetForeignKeys(context.Background(), "test_table")
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetIndexes_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetIndexes(context.Background(), "test_table")
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetConstraints_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetConstraints(context.Background(), "test_table")
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetDatabaseName_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetDatabaseName(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetDatabaseVersion_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetDatabaseVersion(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetSchemaNames_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetSchemaNames(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetTableRowCount_NoConnection(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetTableRowCount(context.Background(), "test_table")
	if err == nil || !strings.Contains(err.Error(), "no active database connection") {
		t.Errorf("expected error, got: %v", err)
	}
}

// --- Not-implemented stubs ---

func TestSampleTableData_NotImplemented(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.SampleTableData(context.Background(), "test", 10)
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' error, got: %v", err)
	}
}

func TestAnalyzeColumnStatistics_NotImplemented(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.AnalyzeColumnStatistics(context.Background(), "test", "col")
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetTableSize_NotImplemented(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetTableSize(context.Background(), "test")
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestGetQueryExecutionPlan_NotImplemented(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.GetQueryExecutionPlan(context.Background(), "SELECT 1")
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestValidatePermissions_NotImplemented(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	err := repo.ValidatePermissions(context.Background(), []string{"SELECT"})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error, got: %v", err)
	}
}

func TestCheckUserPrivileges_NotImplemented(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	_, err := repo.CheckUserPrivileges(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected error, got: %v", err)
	}
}

// --- GetConnectionString tests ---

func TestGetConnectionString_ValidMSSQLConfig(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeMSSQL,
		MSSQL: &models.MSSQLConfig{
			Host:     "dbhost",
			Port:     1433,
			Username: "sa",
			Password: "secret",
			Database: "testdb",
		},
	}

	connStr := repo.GetConnectionString(cfg)
	if !strings.Contains(connStr, "sqlserver://") {
		t.Errorf("expected sqlserver:// prefix, got %q", connStr)
	}
	if !strings.Contains(connStr, "dbhost:1433") {
		t.Errorf("expected host:port in conn string, got %q", connStr)
	}
	if !strings.Contains(connStr, "database=testdb") {
		t.Errorf("expected database in conn string, got %q", connStr)
	}
}

func TestGetConnectionString_WrongConfigType(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type:  models.DatabaseTypeMySQL,
		MySQL: &models.MySQLConfig{Host: "localhost"},
	}

	connStr := repo.GetConnectionString(cfg)
	if connStr != "" {
		t.Errorf("expected empty string for non-MSSQL config, got %q", connStr)
	}
}

func TestGetConnectionString_NilMSSQLConfig(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	cfg := models.DatabaseConfig{Type: models.DatabaseTypeMSSQL}

	connStr := repo.GetConnectionString(cfg)
	if connStr != "" {
		t.Errorf("expected empty string for nil MSSQL config, got %q", connStr)
	}
}

// --- NewMSSQLDatabaseRepository tests ---

func TestNewMSSQLDatabaseRepository(t *testing.T) {
	repo := NewMSSQLDatabaseRepository()
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	mssqlRepo, ok := repo.(*MSSQLDatabaseRepository)
	if !ok {
		t.Fatal("expected *MSSQLDatabaseRepository type")
	}
	if mssqlRepo.db != nil {
		t.Error("expected nil db on new repository")
	}
}

// --- Connect with invalid config tests ---

func TestConnect_InvalidConfigType(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type:  models.DatabaseTypeMySQL,
		MySQL: &models.MySQLConfig{Host: "localhost"},
	}

	_, err := repo.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for non-MSSQL config")
	}
	if !strings.Contains(err.Error(), "expected MSSQLConfig") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConnect_InvalidMSSQLConfig(t *testing.T) {
	repo := &MSSQLDatabaseRepository{}
	cfg := models.DatabaseConfig{
		Type: models.DatabaseTypeMSSQL,
		MSSQL: &models.MSSQLConfig{
			// Missing required fields
			Host: "localhost",
		},
	}

	_, err := repo.Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid MSSQL config")
	}
	if !strings.Contains(err.Error(), "invalid MSSQL configuration") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- applyTableFiltering tests ---

func TestApplyTableFiltering_NoFilters(t *testing.T) {
	tables := []string{"Users", "Orders", "Products"}
	result := applyTableFiltering(tables, models.DataFilteringConfig{})
	if len(result) != 3 {
		t.Errorf("expected 3 tables, got %d", len(result))
	}
}

func TestApplyTableFiltering_Whitelist(t *testing.T) {
	tables := []string{"Users", "Orders", "Products", "Logs"}
	filters := models.DataFilteringConfig{
		TableWhitelist: []string{"Users", "Orders"},
	}
	result := applyTableFiltering(tables, filters)
	if len(result) != 2 {
		t.Errorf("expected 2 tables, got %d", len(result))
	}
}

func TestApplyTableFiltering_Blacklist(t *testing.T) {
	tables := []string{"Users", "Orders", "Products", "Logs"}
	filters := models.DataFilteringConfig{
		TableBlacklist: []string{"Logs"},
	}
	result := applyTableFiltering(tables, filters)
	if len(result) != 3 {
		t.Errorf("expected 3 tables, got %d", len(result))
	}
}

func TestApplyTableFiltering_CaseInsensitive(t *testing.T) {
	tables := []string{"users", "ORDERS"}
	filters := models.DataFilteringConfig{
		TableWhitelist: []string{"Users", "Orders"},
	}
	result := applyTableFiltering(tables, filters)
	if len(result) != 2 {
		t.Errorf("expected 2 tables (case insensitive), got %d", len(result))
	}
}

// --- MSSQLConfig Validate tests ---

func TestMSSQLConfig_Validate_Valid(t *testing.T) {
	cfg := &models.MSSQLConfig{
		Host:     "localhost",
		Username: "sa",
		Password: "secret",
		Database: "testdb",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if cfg.Port != 1433 {
		t.Errorf("expected default port 1433, got %d", cfg.Port)
	}
	if cfg.Schema != "dbo" {
		t.Errorf("expected default schema dbo, got %s", cfg.Schema)
	}
}

func TestMSSQLConfig_Validate_MissingUsername(t *testing.T) {
	cfg := &models.MSSQLConfig{Host: "localhost", Password: "x", Database: "db"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "username") {
		t.Errorf("expected username error, got: %v", err)
	}
}

func TestMSSQLConfig_Validate_MissingHost(t *testing.T) {
	cfg := &models.MSSQLConfig{Username: "sa", Password: "x", Database: "db"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Errorf("expected host error, got: %v", err)
	}
}

func TestMSSQLConfig_BuildConnectionString(t *testing.T) {
	cfg := &models.MSSQLConfig{
		Host:                   "myserver",
		Port:                   1433,
		Username:               "sa",
		Password:               "pass",
		Database:               "mydb",
		Encrypt:                "true",
		TrustServerCertificate: true,
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, "sqlserver://sa:pass@myserver:1433") {
		t.Errorf("unexpected conn string: %s", connStr)
	}
	if !strings.Contains(connStr, "database=mydb") {
		t.Errorf("missing database: %s", connStr)
	}
	if !strings.Contains(connStr, "encrypt=true") {
		t.Errorf("missing encrypt: %s", connStr)
	}
	if !strings.Contains(connStr, "TrustServerCertificate=true") {
		t.Errorf("missing TrustServerCertificate: %s", connStr)
	}
}

func TestMSSQLConfig_BuildConnectionString_WithInstance(t *testing.T) {
	cfg := &models.MSSQLConfig{
		Host:     "myserver",
		Port:     1433,
		Instance: "SQLEXPRESS",
		Username: "sa",
		Password: "pass",
		Database: "mydb",
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, `myserver\SQLEXPRESS:1433`) {
		t.Errorf("expected named instance in conn string, got: %s", connStr)
	}
}
