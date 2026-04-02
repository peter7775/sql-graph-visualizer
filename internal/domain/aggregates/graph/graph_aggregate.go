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

// Package graph contains domain aggregates for graph data structures.
package graph

import (
	"fmt"
	"sql-graph-visualizer/internal/domain/entities"
	"sql-graph-visualizer/internal/domain/events"
	"sql-graph-visualizer/internal/domain/valueobjects/transform"

	"github.com/sirupsen/logrus"
)

// GraphAggregate represents a graph domain aggregate containing nodes and relationships.
//nolint:revive // GraphAggregate is a legacy name we cannot change without breaking existing code
type GraphAggregate struct {
	entities.BaseEntity
	nodes         []*entities.Node
	events        []events.DomainEvent
	relationships []Relationship
}

// Relationship represents a connection between two graph nodes.
type Relationship struct {
	Type       string
	Direction  transform.Direction
	SourceNode *entities.Node
	TargetNode *entities.Node
	Properties map[string]any
}

// NewGraphAggregate creates a new graph aggregate with the given ID.
func NewGraphAggregate(id string) *GraphAggregate {
	return &GraphAggregate{
		BaseEntity: entities.BaseEntity{ID: id},
		nodes:      make([]*entities.Node, 0),
		events:     make([]events.DomainEvent, 0),
	}
}

// AddNode adds a new node to the graph aggregate.
func (g *GraphAggregate) AddNode(nodeType string, properties map[string]any) error {
	existingNode := g.findNode(nodeType, properties["id"], "id")
	if existingNode != nil {
		existingNode.Properties = properties
		return nil
	}

	node := entities.NewNodeWithType(fmt.Sprintf("%s_%v", nodeType, properties["id"]), nodeType, properties["id"], "id")
	node.Properties = properties
	g.nodes = append(g.nodes, node)
	g.events = append(g.events, events.NewNodeAddedEvent(g.ID, node.ID))
	logrus.Infof("Adding node: type=%s, properties=%+v", nodeType, properties)
	return nil
}

// GetNodes returns all nodes in the graph aggregate.
func (g *GraphAggregate) GetNodes() []*entities.Node {
	return g.nodes
}

// GetUncommittedEvents returns all uncommitted domain events.
func (g *GraphAggregate) GetUncommittedEvents() []events.DomainEvent {
	return g.events
}

// ClearEvents clears all uncommitted domain events.
func (g *GraphAggregate) ClearEvents() {
	g.events = []events.DomainEvent{}
}

// AddRelationship adds a relationship between two nodes in the graph.
func (g *GraphAggregate) AddRelationship(
	relType string,
	direction transform.Direction,
	sourceType string,
	sourceKey any,
	sourceField string,
	targetType string,
	targetKey any,
	targetField string,
	properties map[string]any,
) error {
	sourceNode := g.findNode(sourceType, sourceKey, sourceField)
	targetNode := g.findNode(targetType, targetKey, targetField)

	if sourceNode == nil || targetNode == nil {
		logrus.Warnf("Could not find nodes for relationship: source=%s/%v target=%s/%v", sourceType, sourceKey, targetType, targetKey)
		return fmt.Errorf("source or target node not found")
	}

	rel := Relationship{
		Type:       relType,
		Direction:  direction,
		SourceNode: sourceNode,
		TargetNode: targetNode,
		Properties: properties,
	}

	g.relationships = append(g.relationships, rel)
	return nil
}

// ToCypher converts the graph aggregate to Cypher query format.
func (g *GraphAggregate) ToCypher() string {
	return ""
}

func (g *GraphAggregate) findNode(nodeType string, key any, field string) *entities.Node {
	var keyStr string
	switch v := key.(type) {
	case []uint8:
		keyStr = string(v)
	default:
		keyStr = fmt.Sprintf("%v", key)
	}

	for _, node := range g.nodes {
		if node.Type == nodeType {
			var nodeKeyStr string
			switch v := node.Key.(type) {
			case []uint8:
				nodeKeyStr = string(v)
			default:
				nodeKeyStr = fmt.Sprintf("%v", node.Key)
			}

			if node.Type == nodeType && nodeKeyStr == keyStr && node.Field == field {
				return node
			}
		}
	}
	return nil
}

// GetRelationships returns all relationships in the graph aggregate.
func (g *GraphAggregate) GetRelationships() []Relationship {
	return g.relationships
}

// AddDirectRelationship adds a relationship directly to the graph.
func (g *GraphAggregate) AddDirectRelationship(
	relType string,
	sourceNodeID any,
	targetNodeID any,
	properties map[string]any,
) error {
	var sourceNode, targetNode *entities.Node

	for _, node := range g.nodes {
		if node.Properties != nil {
			if nodeID, exists := node.Properties["id"]; exists && fmt.Sprintf("%v", nodeID) == fmt.Sprintf("%v", sourceNodeID) {
				sourceNode = node
			}
			if nodeID, exists := node.Properties["id"]; exists && fmt.Sprintf("%v", nodeID) == fmt.Sprintf("%v", targetNodeID) {
				targetNode = node
			}
		}
	}

	if sourceNode == nil || targetNode == nil {
		logrus.Warnf("Could not find nodes for relationship %s: source=%v, target=%v", relType, sourceNodeID, targetNodeID)
		return fmt.Errorf("source or target node not found for relationship %s", relType)
	}

	rel := Relationship{
		Type:       relType,
		Direction:  transform.Outgoing,
		SourceNode: sourceNode,
		TargetNode: targetNode,
		Properties: properties,
	}

	g.relationships = append(g.relationships, rel)
	logrus.Debugf("Added direct relationship: %s from %v to %v", relType, sourceNodeID, targetNodeID)
	return nil
}
