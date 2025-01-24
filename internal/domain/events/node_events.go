package events

import "time"

type NodeAddedEvent struct {
	BaseDomainEvent
	NodeID string
}

func NewNodeAddedEvent(aggregateID string, nodeID string) *NodeAddedEvent {
	return &NodeAddedEvent{
		BaseDomainEvent: BaseDomainEvent{
			AggregateID: aggregateID,
			EventType:   "NodeAdded",
			OccurredOn:  time.Now(),
		},
		NodeID: nodeID,
	}
}
