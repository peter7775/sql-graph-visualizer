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

// Package neo4j provides Neo4j database persistence configuration.
package neo4j

// Neo4jConfig represents Neo4j database configuration.
//
//nolint:revive // Neo4jConfig is consistent with package naming
type Neo4jConfig struct {
	URI      string
	User     string
	Password string
}
