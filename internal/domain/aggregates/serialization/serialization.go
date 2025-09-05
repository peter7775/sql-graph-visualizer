/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package serialization

import "fmt"

// SerializeID converts an ID to a string format or an array if it's a long type.
func SerializeID(id any) any {
	switch v := id.(type) {
	case int64:
		return []int64{v}
	default:
		return fmt.Sprintf("%v", id)
	}
}
