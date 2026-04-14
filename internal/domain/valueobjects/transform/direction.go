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

// Package transform contains value objects for data transformation rules.
package transform

// Direction represents the direction of a relationship in graph transformations.
type Direction int

const (
	// Outgoing represents an outgoing relationship direction
	Outgoing Direction = iota
	// Incoming represents an incoming relationship direction
	Incoming
	// Both represents bidirectional relationships
	Both
)

func (d Direction) String() string {
	switch d {
	case Outgoing:
		return "OUTGOING"
	case Incoming:
		return "INCOMING"
	case Both:
		return "BOTH"
	default:
		return "UNKNOWN"
	}
}

// ToCypherDirection converts the Direction to Cypher query format.
func (d Direction) ToCypherDirection() string {
	switch d {
	case Outgoing:
		return "->"
	case Incoming:
		return "<-"
	case Both:
		return "-"
	default:
		return "->"
	}
}
