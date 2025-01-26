/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

import (
	"context"
	"fmt"
	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	"mysql-graph-visualizer/internal/domain/valueobjects/transform"

	"github.com/sirupsen/logrus"
)

type TransformService struct {
	mysqlPort ports.MySQLPort
	neo4jPort ports.Neo4jPort
	ruleRepo  ports.TransformRuleRepository
}

func NewTransformService(
	mysqlPort ports.MySQLPort,
	neo4jPort ports.Neo4jPort,
	ruleRepo ports.TransformRuleRepository,
) *TransformService {
	return &TransformService{
		mysqlPort: mysqlPort,
		neo4jPort: neo4jPort,
		ruleRepo:  ruleRepo,
	}
}

func (s *TransformService) TransformAndStore(ctx context.Context) error {
	data, err := s.mysqlPort.FetchData()
	if err != nil {
		return err
	}

	logrus.Infof("Načteno %d záznamů z MySQL", len(data))

	rules, err := s.ruleRepo.GetAllRules(ctx)
	logrus.Infof("Pravidla: %+v", rules)
	if err != nil {
		return err
	}

	graphAggregate := graph.NewGraphAggregate("")

	// Seskupíme data podle tabulek
	tableData := make(map[string][]map[string]interface{})
	for _, item := range data {
		if tableName, ok := item["_table"].(string); ok {
			tableData[tableName] = append(tableData[tableName], item)
		}
	}

	// Načítání uzlů z Neo4j
	nodePHPActionNodes, err := s.neo4jPort.FetchNodes("NodePHPAction")
	if err != nil {
		return fmt.Errorf("chyba při načítání uzlů NodePHPAction z Neo4j: %v", err)
	}

	phpActionNodes, err := s.neo4jPort.FetchNodes("PHPAction")
	if err != nil {
		return fmt.Errorf("chyba při načítání uzlů PHPAction z Neo4j: %v", err)
	}

	// Použití uzlů pro relace
	for _, rule := range rules {
		if rule.Rule.RuleType == transform.RelationshipRule {
			logrus.Infof("Zpracovávám pravidlo pro relaci: %+v", rule)
			transformedData := rule.ApplyRules(append(nodePHPActionNodes, phpActionNodes...))
			logrus.Infof("Transformováno %d záznamů", len(transformedData))
			for _, item := range transformedData {
				if err := s.updateGraph(item, graphAggregate); err != nil {
					return err
				}
			}
		} else if rule.Rule.SourceSQL != "" && rule.Rule.RuleType != transform.RelationshipRule {
			logrus.Infof("Vykonávám SQL dotaz: %s", rule.Rule.SourceSQL)
			items, err := s.mysqlPort.ExecuteQuery(rule.Rule.SourceSQL)
			if err != nil {
				return fmt.Errorf("chyba při vykonávání SQL dotazu: %v", err)
			}
			logrus.Infof("Data vrácená SQL dotazem: %+v", items)
			transformedData := rule.ApplyRules(items)
			logrus.Infof("Transformováno %d záznamů", len(transformedData))
			for _, item := range transformedData {
				if err := s.updateGraph(item, graphAggregate); err != nil {
					return err
				}
			}
		} else {
			sourceTable := rule.Rule.SourceTable
			logrus.Infof("Aplikuji pravidlo na tabulku: %s", sourceTable)
			items, ok := tableData[sourceTable]
			if !ok {
				items = []map[string]interface{}{}
			}
			transformedData := rule.ApplyRules(items)
			logrus.Infof("Transformováno %d záznamů", len(transformedData))
			for _, item := range transformedData {
				if err := s.updateGraph(item, graphAggregate); err != nil {
					return err
				}
			}
		}
	}

	logrus.Infof("Počet uzlů k uložení: %d", len(graphAggregate.GetNodes()))
	logrus.Infof("Ukládám graf do Neo4j")
	return s.neo4jPort.StoreGraph(graphAggregate)
}

func (s *TransformService) updateGraph(data interface{}, graph *graph.GraphAggregate) error {
	switch transformed := data.(type) {
	case map[string]interface{}:
		if nodeType, ok := transformed["_type"].(string); ok {
			if _, hasSource := transformed["source"]; hasSource {
				logrus.Infof("Přidávám vztah do grafu: %+v", transformed)
				return s.createRelationship(transformed, graph)
			}
			logrus.Infof("Přidávám uzel do grafu: %+v", transformed)
			if _, hasName := transformed["name"]; !hasName {
				transformed["name"] = "default_name"
			}
			return s.createNode(nodeType, transformed, graph)
		}
	}
	return fmt.Errorf("invalid transform result format")
}

func (s *TransformService) createNode(nodeType string, data map[string]interface{}, graph *graph.GraphAggregate) error {
	// Kontrola, zda má uzel všechny potřebné vlastnosti
	if _, hasID := data["id"]; !hasID {
		return fmt.Errorf("node data missing required 'id' field")
	}
	if _, hasName := data["name"]; !hasName {
		return fmt.Errorf("node data missing required 'name' field")
	}

	delete(data, "_type")
	logrus.Infof("Ukládám uzel do grafu: typ=%s, data=%+v", nodeType, data)
	return graph.AddNode(nodeType, data)
}

func (s *TransformService) createRelationship(data map[string]interface{}, graph *graph.GraphAggregate) error {
	relType := data["_type"].(string)
	direction := data["_direction"].(transform.Direction)
	source := data["source"].(map[string]interface{})
	target := data["target"].(map[string]interface{})
	properties := data["properties"].(map[string]interface{})

	logrus.Infof("Ukládám vztah do grafu: typ=%s, směr=%s, zdroj=%+v, cíl=%+v, vlastnosti=%+v", relType, direction, source, target, properties)

	return graph.AddRelationship(
		relType,
		direction,
		source["type"].(string),
		source["key"],
		source["field"].(string),
		target["type"].(string),
		target["key"],
		target["field"].(string),
		properties,
	)
}
