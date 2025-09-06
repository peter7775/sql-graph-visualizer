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

type Relation struct {
	BaseEntity
	Type       string
	FromNode   *Node
	ToNode     *Node
	Properties map[string]any
}

func NewRelation(id string, typ string, from *Node, to *Node) *Relation {
	return &Relation{
		BaseEntity: BaseEntity{ID: id},
		Type:       typ,
		FromNode:   from,
		ToNode:     to,
		Properties: make(map[string]any),
	}
}
