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

type Graph struct {
	BaseEntity
	Nodes     []*Node
	Relations []*Relation
}

func NewGraph(id string) *Graph {
	return &Graph{
		BaseEntity: BaseEntity{ID: id},
		Nodes:      make([]*Node, 0),
		Relations:  make([]*Relation, 0),
	}
}
