/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mysql

import (
	"database/sql"
	"mysql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) ports.MySQLPort {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) FetchData() ([]map[string]interface{}, error) {
	// FetchData now only returns empty slice - all data is loaded and processed directly in transform service
	// using ExecuteQuery and rules from configrule repository
	logrus.Infof("💾 FetchData called - returning empty slice (data loading moved to transform service)")
	return []map[string]interface{}{}, nil
}

func (r *MySQLRepository) Close() error {
	return r.db.Close()
}

// getMySQLConfig, mysqlConfig and applyTransformation functions removed
// Configuration now handled by configrule repository

func (r *MySQLRepository) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		columnPointers := make([]interface{}, len(columns))
		for i := range columns {
			columnPointers[i] = new(interface{})
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colName := range columns {
			row[colName] = *(columnPointers[i].(*interface{}))
		}

		results = append(results, row)
	}

	return results, nil
}
