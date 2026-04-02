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

package mysql

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" // MySQL driver registration
)

// MySQLConfig represents MySQL database connection configuration.
//nolint:revive // MySQLConfig is consistent with package naming
type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// Client represents a MySQL database client.
type Client struct {
	db *sql.DB
}

// NewMySQLClient creates a new MySQL client with the given configuration.
func NewMySQLClient(config MySQLConfig) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return &Client{db: db}, nil
}

// Close closes the MySQL database connection.
func (c *Client) Close() error {
	return c.db.Close()
}

// FetchData fetches data from the MySQL database.
func (c *Client) FetchData() ([]map[string]any, error) {
	rows, err := c.db.Query("SELECT * FROM your_table")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		columnPointers := make([]any, len(columns))
		for i := range columns {
			columnPointers[i] = new(any)
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colName := range columns {
			row[colName] = *(columnPointers[i].(*any))
		}

		results = append(results, row)
	}

	return results, nil
}
