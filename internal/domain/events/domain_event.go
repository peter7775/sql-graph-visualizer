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

// Package events contains domain event types and handling.
package events

import "time"

// DomainEvent defines the interface for all domain events.
type DomainEvent interface {
	GetAggregateID() string
	GetEventType() string
	GetOccurredOn() time.Time
}

// BaseDomainEvent provides a base implementation for domain events.
type BaseDomainEvent struct {
	AggregateID string
	EventType   string
	OccurredOn  time.Time
}

// GetAggregateID returns the aggregate ID associated with this event.
func (b *BaseDomainEvent) GetAggregateID() string {
	return b.AggregateID
}

// GetEventType returns the type of this domain event.
func (b *BaseDomainEvent) GetEventType() string {
	return b.EventType
}

// GetOccurredOn returns the time when this event occurred.
func (b *BaseDomainEvent) GetOccurredOn() time.Time {
	return b.OccurredOn
}
