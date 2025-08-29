/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package serialization

import (
	"github.com/google/uuid"
)

// GenerateUniqueID generates a unique identifier for a node.
func GenerateUniqueID() string {
	return uuid.New().String()
}
