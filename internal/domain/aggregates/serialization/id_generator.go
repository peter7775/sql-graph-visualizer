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

// Package serialization provides data serialization and ID generation utilities.
package serialization

import (
	"github.com/google/uuid"
)

// GenerateUniqueID generates a unique identifier string.
func GenerateUniqueID() string {
	return uuid.New().String()
}
