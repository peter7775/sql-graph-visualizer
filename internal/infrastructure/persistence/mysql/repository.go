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
	"log"
	"mysql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) ports.MySQLPort {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) FetchData() ([]map[string]any, error) {
	logrus.Infof("💾 FetchData called - returning empty slice (data loading moved to transform service)")
	return []map[string]any{}, nil
}

func (r *MySQLRepository) Close() error {
	return r.db.Close()
}

func (r *MySQLRepository) ExecuteQuery(query string) ([]map[string]any, error) {
	rows, err := r.db.Query(query)
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
