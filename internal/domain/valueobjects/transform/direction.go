/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
