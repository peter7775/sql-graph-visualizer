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

package events

import "time"

type DomainEvent interface {
	GetAggregateID() string
	GetEventType() string
	GetOccurredOn() time.Time
}

type BaseDomainEvent struct {
	AggregateID string
	EventType   string
	OccurredOn  time.Time
}

func (e *BaseDomainEvent) GetAggregateID() string {
	return e.AggregateID
}

func (e *BaseDomainEvent) GetEventType() string {
	return e.EventType
}

func (e *BaseDomainEvent) GetOccurredOn() time.Time {
	return e.OccurredOn
}
