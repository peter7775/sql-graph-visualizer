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


package entities

// BaseEntity provides basic entity functionality with an ID.
type BaseEntity struct {
	ID string
}

// GetID returns the entity's unique identifier.
func (e *BaseEntity) GetID() string {
	return e.ID
}
