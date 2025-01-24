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
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) ports.MySQLPort {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) FetchData() ([]map[string]interface{}, error) {
	// Původní implementace FetchData
	return nil, nil
}

func (r *MySQLRepository) Close() error {
	return r.db.Close()
}
