/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package models

// Graph represents a graph structure with nodes and relationships.
type Graph struct {
	Nodes     []*Node
	Relations []*Relation
}

// Node represents a single node in the graph.
type Node struct {
	Label      string
	Properties map[string]any
}

// Relation represents a relationship between two nodes in the graph.
type Relation struct {
	Type       string
	From       string
	To         string
	Properties map[string]any
}
