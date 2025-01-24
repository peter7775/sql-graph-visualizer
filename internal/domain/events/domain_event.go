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
