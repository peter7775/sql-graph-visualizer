/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
