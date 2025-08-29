/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package models

type Graph struct {
	Nodes     []*Node
	Relations []*Relation
}

type Node struct {
	Label      string
	Properties map[string]interface{}
}

type Relation struct {
	Type       string
	From       string
	To         string
	Properties map[string]interface{}
}
