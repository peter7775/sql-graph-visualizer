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

// Node represents a graph node entity with properties.
type Node struct {
	BaseEntity
	Type       string
	Key        any
	Field      string
	Label      string
	Properties map[string]any
}

// NewNode creates a new node with the given ID and label.
func NewNode(id string, label string) *Node {
	return &Node{
		BaseEntity: BaseEntity{ID: id},
		Label:      label,
		Properties: make(map[string]any),
	}
}

// NewNodeWithType creates a new node with specified type and properties.
func NewNodeWithType(id string, nodeType string, key any, field string) *Node {
	return &Node{
		BaseEntity: BaseEntity{ID: id},
		Type:       nodeType,
		Key:        key,
		Field:      field,
		Properties: make(map[string]any),
	}
}
