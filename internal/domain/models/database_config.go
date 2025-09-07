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
	"fmt"
	"strings"
)

// DatabaseType represents supported database types
type DatabaseType string

const (
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeOracle     DatabaseType = "oracle"
)

// DatabaseSelector represents configuration for choosing database type
type DatabaseSelector struct {
	Type         DatabaseType      `yaml:"type"` // "mysql" or "postgresql"
	MySQL        *MySQLConfig      `yaml:"mysql,omitempty"`
	PostgreSQL   *PostgreSQLConfig `yaml:"postgresql,omitempty"`
	OracleConfig *OracleConfig     `yaml:"oracle,omitempty"`
}

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	Type       DatabaseType      `yaml:"type" json:"type"`
	MySQL      *MySQLConfig      `yaml:"mysql,omitempty" json:"mysql,omitempty"`
	PostgreSQL *PostgreSQLConfig `yaml:"postgresql,omitempty" json:"postgresql,omitempty"`
	Oracle     *OracleConfig     `yaml:"oracle,omitempty" json:"oracle,omitempty"`
}

// MySQLConfig represents MySQL database configuration

// PostgreSQLConfig represents PostgreSQL database configuration
type PostgreSQLConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Username string `yaml:"username" json:"username"` // Alternative field name
	Password string `yaml:"password" json:"password"`
	Database string `yaml:"database" json:"database"`

	SSLConfig        SSLConfig      `yaml:"ssl" json:"ssl"`
	Security         SecurityConfig `yaml:"security" json:"security"`
	StatementTimeout int            `yaml:"statement_timeout" json:"statement_timeout"` // seconds
	ApplicationName  string         `yaml:"application_name" json:"application_name"`
}

// OracleConfig represents Oracle Database configuration
type OracleConfig struct {
	// Connection details
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	ServiceName string `yaml:"service_name" json:"service_name"`
	SID         string `yaml:"sid" json:"sid"`
	DSN         string `yaml:"dsn" json:"dsn"` // TNS connection descriptor

	// Authentication
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`

	// Connection pool settings
	MaxOpenConns    int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime" json:"conn_max_lifetime"` // minutes

	// Timeout settings
	ConnectionTimeout int `yaml:"connection_timeout" json:"connection_timeout"` // seconds
	QueryTimeout      int `yaml:"query_timeout" json:"query_timeout"`           // seconds

	// Oracle specific
	Timezone        string `yaml:"timezone" json:"timezone"`
	WalletLocation  string `yaml:"wallet_location" json:"wallet_location"`
	ApplicationName string `yaml:"application_name" json:"application_name"`

	// Security
	Security SecurityConfig `yaml:"security" json:"security"`
}

// GetActiveConfig returns the active database configuration based on the selected type
func (ds *DatabaseSelector) GetActiveConfig() DatabaseConfig {
	switch ds.Type {
	case DatabaseTypeMySQL:
		if ds.MySQL != nil {
			return DatabaseConfig{
				Type:  DatabaseTypeMySQL,
				MySQL: ds.MySQL,
			}
		}
	case DatabaseTypePostgreSQL:
		if ds.PostgreSQL != nil {
			return DatabaseConfig{
				Type:       DatabaseTypePostgreSQL,
				PostgreSQL: ds.PostgreSQL,
			}
		}
	case DatabaseTypeOracle:
		if ds.OracleConfig != nil {
			return DatabaseConfig{
				Type:   DatabaseTypeOracle,
				Oracle: ds.OracleConfig,
			}
		}
	}

	// Return empty config if no valid configuration is found
	return DatabaseConfig{}
}

// BuildConnectionString creates Oracle connection string
func (c *OracleConfig) BuildConnectionString() string {
	// If DSN is provided, use it directly (TNS descriptor)
	if c.DSN != "" {
		return fmt.Sprintf("oracle://%s:%s@%s", c.Username, c.Password, c.DSN)
	}

	// Build connection string for simple connection
	var connParts []string

	// Use service name if provided, otherwise SID
	if c.ServiceName != "" {
		connParts = append(connParts, fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			c.Username, c.Password, c.Host, c.Port, c.ServiceName))
	} else if c.SID != "" {
		// For SID connections, use different format
		connParts = append(connParts, fmt.Sprintf("oracle://%s:%s@%s:%d/%s?SID=%s",
			c.Username, c.Password, c.Host, c.Port, c.SID, c.SID))
	} else {
		// Default to XE service
		connParts = append(connParts, fmt.Sprintf("oracle://%s:%s@%s:%d/XE",
			c.Username, c.Password, c.Host, c.Port))
	}

	connString := connParts[0]

	// Add additional parameters
	params := make([]string, 0)

	if c.ConnectionTimeout > 0 {
		params = append(params, fmt.Sprintf("TIMEOUT=%d", c.ConnectionTimeout))
	}

	if c.Timezone != "" {
		params = append(params, fmt.Sprintf("TIMEZONE=%s", c.Timezone))
	}

	if c.WalletLocation != "" {
		params = append(params, fmt.Sprintf("WALLET=%s", c.WalletLocation))
	}

	if len(params) > 0 {
		if strings.Contains(connString, "?") {
			connString += "&" + strings.Join(params, "&")
		} else {
			connString += "?" + strings.Join(params, "&")
		}
	}

	return connString
}

// Validate checks if Oracle configuration is valid
func (c *OracleConfig) Validate() error {
	if c.Username == "" {
		return fmt.Errorf("oracle username is required")
	}

	if c.Password == "" {
		return fmt.Errorf("oracle password is required")
	}

	// If DSN is not provided, check basic connection info
	if c.DSN == "" {
		if c.Host == "" {
			return fmt.Errorf("oracle host is required when DSN is not provided")
		}

		if c.Port <= 0 {
			c.Port = 1521 // Default Oracle port
		}

		if c.ServiceName == "" && c.SID == "" {
			c.ServiceName = "XE" // Default to Oracle XE
		}
	}

	// Set defaults
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 10
	}

	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 5
	}

	if c.ConnectionTimeout <= 0 {
		c.ConnectionTimeout = 30
	}

	if c.QueryTimeout <= 0 {
		c.QueryTimeout = 30
	}

	if c.ApplicationName == "" {
		c.ApplicationName = "sql-graph-visualizer"
	}

	// Set security defaults
	if c.Security.MaxConnections <= 0 {
		c.Security.MaxConnections = c.MaxOpenConns
	}

	if c.Security.ConnectionTimeout <= 0 {
		c.Security.ConnectionTimeout = c.ConnectionTimeout
	}

	if c.Security.QueryTimeout <= 0 {
		c.Security.QueryTimeout = c.QueryTimeout
	}

	return nil
}

// GetEffectiveConfig returns the active database config based on type
func (d *DatabaseConfig) GetEffectiveConfig() interface{} {
	switch d.Type {
	case DatabaseTypeMySQL:
		return d.MySQL
	case DatabaseTypePostgreSQL:
		return d.PostgreSQL
	case DatabaseTypeOracle:
		return d.Oracle
	default:
		return nil
	}
}

// Validate validates the database configuration
func (d *DatabaseConfig) Validate() error {
	switch d.Type {
	case DatabaseTypeMySQL:
		if d.MySQL == nil {
			return fmt.Errorf("mysql configuration is required when type is mysql")
		}
		// Add MySQL-specific validation if needed
	case DatabaseTypePostgreSQL:
		if d.PostgreSQL == nil {
			return fmt.Errorf("postgresql configuration is required when type is postgresql")
		}
		// Add PostgreSQL-specific validation if needed
	case DatabaseTypeOracle:
		if d.Oracle == nil {
			return fmt.Errorf("oracle configuration is required when type is oracle")
		}
		return d.Oracle.Validate()
	default:
		return fmt.Errorf("unsupported database type: %s", d.Type)
	}
	return nil
}

// IsSupported checks if the database type is supported
func (dt DatabaseType) IsSupported() bool {
	switch dt {
	case DatabaseTypeMySQL, DatabaseTypePostgreSQL, DatabaseTypeOracle:
		return true
	default:
		return false
	}
}

// String returns string representation of database type
func (dt DatabaseType) String() string {
	return string(dt)
}
