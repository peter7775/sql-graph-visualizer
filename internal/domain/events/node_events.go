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

// NodeAddedEvent represents an event when a node is added to the graph.
type NodeAddedEvent struct {
	BaseDomainEvent
	NodeID string
}

// NewNodeAddedEvent creates a new NodeAddedEvent.
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
