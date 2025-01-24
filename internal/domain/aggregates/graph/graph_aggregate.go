/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package graph

import (
	"fmt"
	"mysql-graph-visualizer/internal/domain/entities"
	"mysql-graph-visualizer/internal/domain/events"
	"mysql-graph-visualizer/internal/domain/valueobjects"
	"mysql-graph-visualizer/internal/domain/valueobjects/transform"
)

type GraphAggregate struct {
	entities.BaseEntity
	nodes         []*entities.Node
	criteria      valueobjects.SearchCriteria
	events        []events.DomainEvent
	relationships []Relationship
}

type Relationship struct {
	Type       string
	Direction  transform.Direction
	SourceNode *entities.Node
	TargetNode *entities.Node
	Properties map[string]interface{}
}

func NewGraphAggregate(id string) *GraphAggregate {
	return &GraphAggregate{
		BaseEntity: entities.BaseEntity{ID: id},
		nodes:      make([]*entities.Node, 0),
		events:     make([]events.DomainEvent, 0),
	}
}

func (g *GraphAggregate) AddNode(nodeType string, properties map[string]interface{}) error {
	// Nejprve zkontrolujeme, zda uzel již neexistuje
	existingNode := g.findNode(nodeType, properties["id"], "id")
	if existingNode != nil {
		// Pokud uzel existuje, aktualizujeme jeho vlastnosti a vrátíme nil
		existingNode.Properties = properties
		return nil
	}

	// Pokud uzel neexistuje, vytvoříme nový
	node := entities.NewNodeWithType(fmt.Sprintf("%s_%v", nodeType, properties["id"]), nodeType, properties["id"], "id")
	node.Properties = properties
	g.nodes = append(g.nodes, node)
	g.events = append(g.events, events.NewNodeAddedEvent(g.ID, node.ID))
	return nil
}

func (g *GraphAggregate) GetNodes() []*entities.Node {
	return g.nodes
}

func (g *GraphAggregate) GetUncommittedEvents() []events.DomainEvent {
	return g.events
}

func (g *GraphAggregate) ClearEvents() {
	g.events = []events.DomainEvent{}
}

func (g *GraphAggregate) AddRelationship(
	relType string,
	direction transform.Direction,
	sourceType string,
	sourceKey interface{},
	sourceField string,
	targetType string,
	targetKey interface{},
	targetField string,
	properties map[string]interface{},
) error {
	sourceNode := g.findNode(sourceType, sourceKey, sourceField)
	targetNode := g.findNode(targetType, targetKey, targetField)

	if sourceNode == nil || targetNode == nil {
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

func (g *GraphAggregate) ToCypher() string {
	// Implementace generování Cypher dotazu pro vytvoření grafu
	// Použije Direction.ToCypherDirection() pro správné směrování relací
	return ""
}

func (g *GraphAggregate) findNode(nodeType string, key interface{}, field string) *entities.Node {
	// Převod klíče na string pro porovnání
	var keyStr string
	switch v := key.(type) {
	case []uint8:
		keyStr = string(v)
	default:
		keyStr = fmt.Sprintf("%v", key)
	}

	for _, node := range g.nodes {
		if node.Type == nodeType {
			// Převod node.Key na string pro porovnání
			var nodeKeyStr string
			switch v := node.Key.(type) {
			case []uint8:
				nodeKeyStr = string(v)
			default:
				nodeKeyStr = fmt.Sprintf("%v", node.Key)
			}

			// Porovnání všech tří podmínek
			if node.Type == nodeType && nodeKeyStr == keyStr && node.Field == field {
				return node
			}
		}
	}
	return nil
}

func (g *GraphAggregate) GetRelationships() []Relationship {
	return g.relationships
}
