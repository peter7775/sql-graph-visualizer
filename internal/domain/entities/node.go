/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package entities

type Node struct {
	BaseEntity
	Type       string
	Key        interface{}
	Field      string
	Label      string
	Properties map[string]interface{}
}

func NewNode(id string, label string) *Node {
	return &Node{
		BaseEntity: BaseEntity{ID: id},
		Label:      label,
		Properties: make(map[string]interface{}),
	}
}

func NewNodeWithType(id string, nodeType string, key interface{}, field string) *Node {
	return &Node{
		BaseEntity: BaseEntity{ID: id},
		Type:       nodeType,
		Key:        key,
		Field:      field,
		Properties: make(map[string]interface{}),
	}
}
