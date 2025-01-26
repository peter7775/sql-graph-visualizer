/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package neo4j

import (
	"fmt"
	"log"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Neo4jRepository struct {
	driver neo4j.Driver
}

func NewNeo4jRepository(uri, username, password string) (*Neo4jRepository, error) {
    driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        return nil, err
    }
    return &Neo4jRepository{driver: driver}, nil
}

func (r *Neo4jRepository) StoreGraph(graph *graph.GraphAggregate) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Uložení uzlů
	for _, node := range graph.GetNodes() {
		query := "CREATE (n:" + node.Type + ") SET n = $props"
		if _, err := session.Run(query, map[string]interface{}{
			"props": node.Properties,
		}); err != nil {
			return err
		}
		log.Printf("Uložen uzel: typ=%s, vlastnosti=%+v", node.Type, node.Properties)
	}

	// Uložení vztahů
	for _, rel := range graph.GetRelationships() {
		query := "MATCH (a {id: $sourceId}), (b {id: $targetId}) CREATE (a)-[r:" + rel.Type + "]->(b) SET r = $props"
		if _, err := session.Run(query, map[string]interface{}{
			"sourceId": rel.SourceNode.ID,
			"targetId": rel.TargetNode.ID,
			"props":    rel.Properties,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *Neo4jRepository) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(criteria, nil)
	if err != nil {
		return nil, err
	}

	if !result.Next() {
		return nil, nil
	}

	record := result.Record()
	count := record.Values[0].(int64)

	// Vytvoříme jeden GraphAggregate s uzly
	graphAgg := graph.NewGraphAggregate("")
	for i := int64(0); i < count; i++ {
		graphAgg.AddNode("Person", map[string]interface{}{
			"id":   i + 1,
			"name": fmt.Sprintf("Person %d", i+1),
		})
	}

	return []*graph.GraphAggregate{graphAgg}, nil
}

func (r *Neo4jRepository) ExportGraph(query string) (interface{}, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(query, nil)
	if err != nil {
		return nil, err
	}

	graphAgg := graph.NewGraphAggregate("")

	for result.Next() {
		record := result.Record()
		node := record.GetByIndex(0).(neo4j.Node)

		// Přidáme uzel do grafu
		graphAgg.AddNode(node.Labels[0], node.Props)
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return graphAgg, nil
}

func (r *Neo4jRepository) Close() error {
	return r.driver.Close()
}
