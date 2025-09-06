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


package transform

type Direction int

const (
	Outgoing Direction = iota
	Incoming
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
