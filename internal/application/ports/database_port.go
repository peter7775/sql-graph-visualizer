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

package ports

// DatabasePort is a generic interface for database operations used by transform services
// This interface abstracts the common database operations needed for data transformation,
// regardless of the underlying database type (MySQL, PostgreSQL, etc.)
type DatabasePort interface {
	FetchData() ([]map[string]any, error)
	ExecuteQuery(query string) ([]map[string]any, error)
	Close() error
}
