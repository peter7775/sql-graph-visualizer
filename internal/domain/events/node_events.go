/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

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
