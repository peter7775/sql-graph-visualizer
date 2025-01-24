package transform

import (
	"github.com/peter7775/alevisualizer/internal/application/ports"
	"github.com/peter7775/alevisualizer/internal/domain/aggregates/graph"
	"github.com/peter7775/alevisualizer/internal/domain/valueobjects/transform"
	"context"
	"fmt"
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

	rules, err := s.ruleRepo.GetAllRules(ctx)
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

	// Aplikujeme pravidla na příslušná data
	for _, rule := range rules {
		sourceTable := rule.Rule.SourceTable
		if items, ok := tableData[sourceTable]; ok {
			transformedData := rule.ApplyRules(items)
			for _, item := range transformedData {
				if err := s.updateGraph(item, graphAggregate); err != nil {
					return err
				}
			}
		}
	}

	return s.neo4jPort.StoreGraph(graphAggregate)
}

func (s *TransformService) updateGraph(data interface{}, graph *graph.GraphAggregate) error {
	switch transformed := data.(type) {
	case map[string]interface{}:
		if nodeType, ok := transformed["_type"].(string); ok {
			if _, hasSource := transformed["source"]; hasSource {
				return s.createRelationship(transformed, graph)
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
	return graph.AddNode(nodeType, data)
}

func (s *TransformService) createRelationship(data map[string]interface{}, graph *graph.GraphAggregate) error {
	relType := data["_type"].(string)
	direction := data["_direction"].(transform.Direction)
	source := data["source"].(map[string]interface{})
	target := data["target"].(map[string]interface{})
	properties := data["properties"].(map[string]interface{})

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
