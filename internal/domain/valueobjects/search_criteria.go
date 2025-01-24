/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package valueobjects

import "fmt"

type SearchCriteria struct {
	Labels     []string
	Properties map[string]interface{}
}

func (c SearchCriteria) ToString() string {
	query := "MATCH (n)"
	if len(c.Labels) > 0 {
		query += fmt.Sprintf(" WHERE n:%s", c.Labels[0])
		for _, label := range c.Labels[1:] {
			query += fmt.Sprintf(" OR n:%s", label)
		}
	}
	return query + " RETURN n LIMIT 100"
}
