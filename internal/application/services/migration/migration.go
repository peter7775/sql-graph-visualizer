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

// Package migration provides database migration services.
package migration

import (
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

// MigrationConfig represents configuration for database migrations.
//nolint:revive // MigrationConfig follows established naming pattern
type MigrationConfig struct {
	SourceTable string
	Neo4jDriver neo4j.Driver
}
