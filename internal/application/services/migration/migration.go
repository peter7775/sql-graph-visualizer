/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package migration

import (
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type MigrationConfig struct {
	SourceTable string
	Neo4jDriver neo4j.Driver
}
