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

package models

import (
	"strings"
	"testing"
)

// --- OracleConfig.Validate() tests ---

func TestOracleConfig_Validate_ValidMinimal(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "localhost",
		Username: "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check defaults were set
	if cfg.Port != 1521 {
		t.Errorf("expected default port 1521, got %d", cfg.Port)
	}
	if cfg.ServiceName != "XE" {
		t.Errorf("expected default service name XE, got %s", cfg.ServiceName)
	}
	if cfg.MaxOpenConns != 10 {
		t.Errorf("expected default MaxOpenConns 10, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("expected default MaxIdleConns 5, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnectionTimeout != 30 {
		t.Errorf("expected default ConnectionTimeout 30, got %d", cfg.ConnectionTimeout)
	}
	if cfg.QueryTimeout != 30 {
		t.Errorf("expected default QueryTimeout 30, got %d", cfg.QueryTimeout)
	}
	if cfg.ApplicationName != "sql-graph-visualizer" {
		t.Errorf("expected default ApplicationName, got %s", cfg.ApplicationName)
	}
}

func TestOracleConfig_Validate_MissingUsername(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "localhost",
		Password: "testpass",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing username")
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOracleConfig_Validate_MissingPassword(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "localhost",
		Username: "testuser",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing password")
	}
	if !strings.Contains(err.Error(), "password is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOracleConfig_Validate_MissingHostWithoutDSN(t *testing.T) {
	cfg := &OracleConfig{
		Username: "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing host")
	}
	if !strings.Contains(err.Error(), "host is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOracleConfig_Validate_DSNBypassesHostCheck(t *testing.T) {
	cfg := &OracleConfig{
		DSN:      "(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db.example.com)(PORT=1521))(CONNECT_DATA=(SID=ORCL)))",
		Username: "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error with DSN, got: %v", err)
	}
}

func TestOracleConfig_Validate_ExplicitPortPreserved(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "localhost",
		Port:     1522,
		Username: "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 1522 {
		t.Errorf("expected port 1522 to be preserved, got %d", cfg.Port)
	}
}

func TestOracleConfig_Validate_ExplicitServiceNamePreserved(t *testing.T) {
	cfg := &OracleConfig{
		Host:        "localhost",
		Username:    "testuser",
		Password:    "testpass",
		ServiceName: "ORCL",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ServiceName != "ORCL" {
		t.Errorf("expected service name ORCL, got %s", cfg.ServiceName)
	}
}

func TestOracleConfig_Validate_SIDPreventDefaultServiceName(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "localhost",
		Username: "testuser",
		Password: "testpass",
		SID:      "MYSID",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When SID is set, ServiceName should NOT be defaulted to XE
	if cfg.ServiceName != "" {
		t.Errorf("expected empty service name when SID is set, got %s", cfg.ServiceName)
	}
}

func TestOracleConfig_Validate_SecurityDefaults(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "localhost",
		Username: "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Security.MaxConnections != cfg.MaxOpenConns {
		t.Errorf("expected Security.MaxConnections=%d, got %d", cfg.MaxOpenConns, cfg.Security.MaxConnections)
	}
	if cfg.Security.ConnectionTimeout != cfg.ConnectionTimeout {
		t.Errorf("expected Security.ConnectionTimeout=%d, got %d", cfg.ConnectionTimeout, cfg.Security.ConnectionTimeout)
	}
	if cfg.Security.QueryTimeout != cfg.QueryTimeout {
		t.Errorf("expected Security.QueryTimeout=%d, got %d", cfg.QueryTimeout, cfg.Security.QueryTimeout)
	}
}

func TestOracleConfig_Validate_ExplicitValuesNotOverwritten(t *testing.T) {
	cfg := &OracleConfig{
		Host:              "dbhost",
		Port:              1523,
		ServiceName:       "PROD",
		Username:          "admin",
		Password:          "secret",
		MaxOpenConns:      20,
		MaxIdleConns:      10,
		ConnectionTimeout: 60,
		QueryTimeout:      120,
		ApplicationName:   "my-app",
		Security: SecurityConfig{
			MaxConnections:    15,
			ConnectionTimeout: 45,
			QueryTimeout:      90,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MaxOpenConns != 20 {
		t.Errorf("MaxOpenConns should not be overwritten, got %d", cfg.MaxOpenConns)
	}
	if cfg.ApplicationName != "my-app" {
		t.Errorf("ApplicationName should not be overwritten, got %s", cfg.ApplicationName)
	}
	if cfg.Security.MaxConnections != 15 {
		t.Errorf("Security.MaxConnections should not be overwritten, got %d", cfg.Security.MaxConnections)
	}
}

// --- OracleConfig.BuildConnectionString() tests ---

func TestOracleConfig_BuildConnectionString_ServiceName(t *testing.T) {
	cfg := &OracleConfig{
		Host:        "dbhost",
		Port:        1521,
		ServiceName: "ORCL",
		Username:    "admin",
		Password:    "secret",
	}

	connStr := cfg.BuildConnectionString()
	expected := "oracle://admin:secret@dbhost:1521/ORCL"
	if connStr != expected {
		t.Errorf("expected %q, got %q", expected, connStr)
	}
}

func TestOracleConfig_BuildConnectionString_SID(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "dbhost",
		Port:     1521,
		SID:      "MYSID",
		Username: "admin",
		Password: "secret",
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, "MYSID") {
		t.Errorf("expected connection string to contain SID, got %q", connStr)
	}
	if !strings.Contains(connStr, "SID=MYSID") {
		t.Errorf("expected SID parameter in connection string, got %q", connStr)
	}
}

func TestOracleConfig_BuildConnectionString_DSN(t *testing.T) {
	cfg := &OracleConfig{
		DSN:      "my-tns-descriptor",
		Username: "admin",
		Password: "secret",
	}

	connStr := cfg.BuildConnectionString()
	expected := "oracle://admin:secret@my-tns-descriptor"
	if connStr != expected {
		t.Errorf("expected %q, got %q", expected, connStr)
	}
}

func TestOracleConfig_BuildConnectionString_DefaultXE(t *testing.T) {
	cfg := &OracleConfig{
		Host:     "dbhost",
		Port:     1521,
		Username: "admin",
		Password: "secret",
	}

	connStr := cfg.BuildConnectionString()
	expected := "oracle://admin:secret@dbhost:1521/XE"
	if connStr != expected {
		t.Errorf("expected %q, got %q", expected, connStr)
	}
}

func TestOracleConfig_BuildConnectionString_WithTimeout(t *testing.T) {
	cfg := &OracleConfig{
		Host:              "dbhost",
		Port:              1521,
		ServiceName:       "ORCL",
		Username:          "admin",
		Password:          "secret",
		ConnectionTimeout: 60,
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, "TIMEOUT=60") {
		t.Errorf("expected TIMEOUT parameter, got %q", connStr)
	}
}

func TestOracleConfig_BuildConnectionString_WithTimezone(t *testing.T) {
	cfg := &OracleConfig{
		Host:        "dbhost",
		Port:        1521,
		ServiceName: "ORCL",
		Username:    "admin",
		Password:    "secret",
		Timezone:    "UTC",
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, "TIMEZONE=UTC") {
		t.Errorf("expected TIMEZONE parameter, got %q", connStr)
	}
}

func TestOracleConfig_BuildConnectionString_WithWallet(t *testing.T) {
	cfg := &OracleConfig{
		Host:           "dbhost",
		Port:           1521,
		ServiceName:    "ORCL",
		Username:       "admin",
		Password:       "secret",
		WalletLocation: "/opt/wallet",
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, "WALLET=/opt/wallet") {
		t.Errorf("expected WALLET parameter, got %q", connStr)
	}
}

func TestOracleConfig_BuildConnectionString_MultipleParams(t *testing.T) {
	cfg := &OracleConfig{
		Host:              "dbhost",
		Port:              1521,
		ServiceName:       "ORCL",
		Username:          "admin",
		Password:          "secret",
		ConnectionTimeout: 60,
		Timezone:          "UTC",
	}

	connStr := cfg.BuildConnectionString()
	if !strings.Contains(connStr, "TIMEOUT=60") {
		t.Errorf("expected TIMEOUT parameter, got %q", connStr)
	}
	if !strings.Contains(connStr, "TIMEZONE=UTC") {
		t.Errorf("expected TIMEZONE parameter, got %q", connStr)
	}
	if !strings.Contains(connStr, "&") {
		t.Errorf("expected parameters joined with &, got %q", connStr)
	}
}

func TestOracleConfig_BuildConnectionString_SIDWithParams(t *testing.T) {
	cfg := &OracleConfig{
		Host:              "dbhost",
		Port:              1521,
		SID:               "MYSID",
		Username:          "admin",
		Password:          "secret",
		ConnectionTimeout: 30,
	}

	connStr := cfg.BuildConnectionString()
	// SID URL already contains ?, so params should use &
	if !strings.Contains(connStr, "SID=MYSID") {
		t.Errorf("expected SID parameter, got %q", connStr)
	}
	if !strings.Contains(connStr, "TIMEOUT=30") {
		t.Errorf("expected TIMEOUT parameter, got %q", connStr)
	}
	if !strings.Contains(connStr, "&TIMEOUT") {
		t.Errorf("expected & separator for additional params with SID, got %q", connStr)
	}
}

// --- DatabaseConfig Oracle methods tests ---

func TestDatabaseConfig_GetHost_Oracle(t *testing.T) {
	cfg := DatabaseConfig{
		Type:   DatabaseTypeOracle,
		Oracle: &OracleConfig{Host: "oracle-host"},
	}

	if cfg.GetHost() != "oracle-host" {
		t.Errorf("expected oracle-host, got %s", cfg.GetHost())
	}
}

func TestDatabaseConfig_GetHost_Oracle_Nil(t *testing.T) {
	cfg := DatabaseConfig{
		Type: DatabaseTypeOracle,
	}

	if cfg.GetHost() != "" {
		t.Errorf("expected empty string for nil Oracle config, got %s", cfg.GetHost())
	}
}

func TestDatabaseConfig_GetPort_Oracle(t *testing.T) {
	cfg := DatabaseConfig{
		Type:   DatabaseTypeOracle,
		Oracle: &OracleConfig{Port: 1521},
	}

	if cfg.GetPort() != 1521 {
		t.Errorf("expected 1521, got %d", cfg.GetPort())
	}
}

func TestDatabaseConfig_GetPort_Oracle_Nil(t *testing.T) {
	cfg := DatabaseConfig{
		Type: DatabaseTypeOracle,
	}

	if cfg.GetPort() != 0 {
		t.Errorf("expected 0 for nil Oracle config, got %d", cfg.GetPort())
	}
}

func TestDatabaseConfig_GetUsername_Oracle(t *testing.T) {
	cfg := DatabaseConfig{
		Type:   DatabaseTypeOracle,
		Oracle: &OracleConfig{Username: "orauser"},
	}

	if cfg.GetUsername() != "orauser" {
		t.Errorf("expected orauser, got %s", cfg.GetUsername())
	}
}

func TestDatabaseConfig_GetUsername_Oracle_Nil(t *testing.T) {
	cfg := DatabaseConfig{
		Type: DatabaseTypeOracle,
	}

	if cfg.GetUsername() != "" {
		t.Errorf("expected empty string for nil Oracle config, got %s", cfg.GetUsername())
	}
}

func TestDatabaseConfig_GetEffectiveConfig_Oracle(t *testing.T) {
	oracleCfg := &OracleConfig{Host: "oracle-host"}
	cfg := DatabaseConfig{
		Type:   DatabaseTypeOracle,
		Oracle: oracleCfg,
	}

	result := cfg.GetEffectiveConfig()
	if result != oracleCfg {
		t.Error("expected GetEffectiveConfig to return Oracle config")
	}
}

func TestDatabaseConfig_GetEffectiveConfig_Unsupported(t *testing.T) {
	cfg := DatabaseConfig{
		Type: "unknown",
	}

	if cfg.GetEffectiveConfig() != nil {
		t.Error("expected nil for unsupported type")
	}
}

func TestDatabaseConfig_GetDatabaseType_Oracle(t *testing.T) {
	cfg := DatabaseConfig{Type: DatabaseTypeOracle}
	if cfg.GetDatabaseType() != DatabaseTypeOracle {
		t.Errorf("expected oracle, got %s", cfg.GetDatabaseType())
	}
}

// --- DatabaseConfig.Validate() tests ---

func TestDatabaseConfig_Validate_Oracle_Valid(t *testing.T) {
	cfg := DatabaseConfig{
		Type: DatabaseTypeOracle,
		Oracle: &OracleConfig{
			Host:     "localhost",
			Username: "admin",
			Password: "secret",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDatabaseConfig_Validate_Oracle_NilConfig(t *testing.T) {
	cfg := DatabaseConfig{
		Type: DatabaseTypeOracle,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Oracle config")
	}
	if !strings.Contains(err.Error(), "oracle configuration is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDatabaseConfig_Validate_UnsupportedType(t *testing.T) {
	cfg := DatabaseConfig{
		Type: "sqlite",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported database type") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- DatabaseSelector.GetActiveConfig() tests ---

func TestDatabaseSelector_GetActiveConfig_Oracle(t *testing.T) {
	oracleCfg := &OracleConfig{Host: "oracle-host", Username: "orauser"}
	selector := &DatabaseSelector{
		Type:         DatabaseTypeOracle,
		OracleConfig: oracleCfg,
	}

	config := selector.GetActiveConfig()
	if config.Type != DatabaseTypeOracle {
		t.Errorf("expected oracle type, got %s", config.Type)
	}
	if config.Oracle != oracleCfg {
		t.Error("expected Oracle config to match")
	}
}

func TestDatabaseSelector_GetActiveConfig_Oracle_Nil(t *testing.T) {
	selector := &DatabaseSelector{
		Type: DatabaseTypeOracle,
	}

	config := selector.GetActiveConfig()
	if config.Type != "" {
		t.Errorf("expected empty config for nil Oracle, got type %s", config.Type)
	}
}

// --- DatabaseType tests ---

func TestDatabaseType_IsSupported_Oracle(t *testing.T) {
	if !DatabaseTypeOracle.IsSupported() {
		t.Error("Oracle should be supported")
	}
}

func TestDatabaseType_IsSupported_Unknown(t *testing.T) {
	dt := DatabaseType("sqlite")
	if dt.IsSupported() {
		t.Error("sqlite should not be supported")
	}
}

func TestDatabaseType_String_Oracle(t *testing.T) {
	if DatabaseTypeOracle.String() != "oracle" {
		t.Errorf("expected 'oracle', got %s", DatabaseTypeOracle.String())
	}
}
