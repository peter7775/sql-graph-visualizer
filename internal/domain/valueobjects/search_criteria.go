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

package valueobjects

import "fmt"

type SearchCriteria struct {
	Labels     []string
	Properties map[string]any
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
