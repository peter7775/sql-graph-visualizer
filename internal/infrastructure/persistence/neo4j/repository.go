/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package neo4j

import (
	"fmt"
	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Neo4jRepository struct {
	driver neo4j.Driver
}

func NewNeo4jRepository(driver neo4j.Driver) ports.Neo4jPort {
	return &Neo4jRepository{driver: driver}
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
	}

	// Uložení vztahů - zatím přeskočíme, protože nemáme implementovaný přístup k vztahům
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

	if result.Next() {
		return result.Record().GetByIndex(0), nil
	}

	return nil, nil
}

func (r *Neo4jRepository) Close() error {
	return r.driver.Close()
}
