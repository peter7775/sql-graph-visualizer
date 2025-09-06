/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
