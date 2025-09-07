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
