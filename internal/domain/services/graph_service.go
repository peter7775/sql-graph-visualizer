package services

import (
	"github.com/peter7775/alevisualizer/internal/domain/models"
	"github.com/peter7775/alevisualizer/internal/domain/repositories"
)

type GraphService interface {
	SearchNodes(term string) ([]models.SearchResult, error)
	ExportImage() ([]byte, error)
	ExportJSON() (interface{}, error)
}

type Neo4jGraphService struct {
	repo repositories.Neo4jRepository
}

func NewNeo4jGraphService(repo repositories.Neo4jRepository) GraphService {
	return &Neo4jGraphService{repo: repo}
}

func (s *Neo4jGraphService) SearchNodes(term string) ([]models.SearchResult, error) {
	query := `
		MATCH (n)
		WHERE n.name =~ $term OR n.email =~ $term
		RETURN n.id as id, n.name as name, labels(n) as labels
		LIMIT 10
	`
	return s.repo.SearchNodes(query, map[string]interface{}{
		"term": "(?i).*" + term + ".*",
	})
}

func (s *Neo4jGraphService) ExportImage() ([]byte, error) {
	// Implementace exportu do PNG
	// Můžete použít knihovnu jako je "github.com/fogleman/gg" pro vykreslení grafu
	return nil, nil
}

func (s *Neo4jGraphService) ExportJSON() (interface{}, error) {
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
