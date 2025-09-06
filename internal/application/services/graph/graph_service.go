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


package graph

import (
	"mysql-graph-visualizer/internal/domain/models"
	"mysql-graph-visualizer/internal/domain/repositories"
)

type GraphService interface {
	SearchNodes(term string) ([]models.SearchResult, error)
	ExportImage() ([]byte, error)
	ExportJSON() (any, error)
}

type Neo4jGraphService struct {
	repo repositories.Neo4jRepository
}

func NewNeo4jGraphService(repo repositories.Neo4jRepository) GraphService {
	return &Neo4jGraphService{repo: repo}
}

func (s *Neo4jGraphService) SearchNodes(term string) ([]models.SearchResult, error) {
	criteria := "(?i).*" + term + ".*"
	graphs, err := s.repo.SearchNodes(criteria)
	if err != nil {
		return nil, err
	}

	// Convert GraphAggregates to SearchResults (placeholder implementation)
	var results []models.SearchResult
	for _, graphAgg := range graphs {
		for _, node := range graphAgg.GetNodes() {
			result := models.SearchResult{
				ID:     node.ID,
				Name:   node.Type, // Using type as name for now
				Labels: []string{node.Type},
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func (s *Neo4jGraphService) ExportImage() ([]byte, error) {
	// Implementation of PNG export
	// You can use a library like "github.com/fogleman/gg" for graph rendering
	return nil, nil
}

func (s *Neo4jGraphService) ExportJSON() (any, error) {
	query := `
		 MATCH (n)-[r]->(m)
		 RETURN {
			 nodes: collect(distinct {
				 id: id(n),
				 labels: labels(n),
				 properties: properties(n)
			 }),
			 relationships: collect({
				 id: id(r),
				 type: type(r),
				 properties: properties(r),
				 source: id(n),
				 target: id(m)
			 })
		 } as graph
	 `
	return s.repo.ExportGraph(query)
}
