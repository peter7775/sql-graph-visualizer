/*
 * SQL Graph Visualizer - Database Schema Analyzer Service
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package services

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
)

// SchemaAnalyzerService provides advanced database schema analysis
// and automatic transformation rule generation capabilities
type SchemaAnalyzerService struct {
	mysqlPort ports.MySQLPort
	config    *models.SchemaAnalysisConfig
}

// NewSchemaAnalyzerService creates a new schema analyzer service instance
func NewSchemaAnalyzerService(mysqlPort ports.MySQLPort, config *models.SchemaAnalysisConfig) *SchemaAnalyzerService {
	return &SchemaAnalyzerService{
		mysqlPort: mysqlPort,
		config:    config,
	}
}

// AnalyzeSchemaFromConnection performs comprehensive schema analysis
// on an existing database connection
func (s *SchemaAnalyzerService) AnalyzeSchemaFromConnection(
	ctx context.Context,
	db *sql.DB,
	filterConfig *models.DataFilteringConfig,
) (*models.SchemaAnalysisResult, error) {
	
	// Step 1: Validate connection
	validation, err := s.mysqlPort.ValidateConnection(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("connection validation failed: %w", err)
	}
	if !validation.IsValid {
		return nil, fmt.Errorf("connection is not valid: %s", validation.ErrorMessage)
	}

	// Step 2: Discover schema structure
	result, err := s.mysqlPort.DiscoverSchema(ctx, db, filterConfig)
	if err != nil {
		return nil, fmt.Errorf("schema discovery failed: %w", err)
	}

	// Step 3: Analyze relationships and patterns
	err = s.enrichSchemaWithRelationshipAnalysis(ctx, db, result)
	if err != nil {
		return nil, fmt.Errorf("relationship analysis failed: %w", err)
	}

	// Step 4: Generate transformation rules
	err = s.generateTransformationRules(result)
	if err != nil {
		return nil, fmt.Errorf("rule generation failed: %w", err)
	}

	// Step 5: Estimate data size and complexity
	datasetInfo, err := s.mysqlPort.EstimateDataSize(ctx, db, filterConfig)
	if err != nil {
		return nil, fmt.Errorf("data size estimation failed: %w", err)
	}
	result.DatasetInfo = datasetInfo

	return result, nil
}

// enrichSchemaWithRelationshipAnalysis analyzes foreign key relationships
// and identifies potential graph patterns
func (s *SchemaAnalyzerService) enrichSchemaWithRelationshipAnalysis(
	ctx context.Context,
	db *sql.DB,
	result *models.SchemaAnalysisResult,
) error {
	
	// Analyze foreign key relationships
	for _, table := range result.Tables {
		relationships, err := s.analyzeForeignKeyRelationships(ctx, db, table.Name)
		if err != nil {
			return fmt.Errorf("failed to analyze relationships for table %s: %w", table.Name, err)
		}
		table.Relationships = relationships

		// Identify potential junction tables (many-to-many relationships)
		if s.isJunctionTable(table) {
			table.GraphType = "RELATIONSHIP"
			table.Recommendations = append(table.Recommendations, 
				"This table appears to be a junction table - consider modeling as Neo4j relationships")
		} else {
			table.GraphType = "NODE"
		}
	}

	// Identify graph patterns
	result.GraphPatterns = s.identifyGraphPatterns(result.Tables)
	
	return nil
}

// analyzeForeignKeyRelationships discovers foreign key constraints
func (s *SchemaAnalyzerService) analyzeForeignKeyRelationships(
	ctx context.Context,
	db *sql.DB,
	tableName string,
) ([]*models.Relationship, error) {
	
	query := `
		SELECT 
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME,
			CONSTRAINT_NAME
		FROM 
			INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
		WHERE 
			TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL
	`

	rows, err := db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []*models.Relationship
	for rows.Next() {
		var rel models.Relationship
		err := rows.Scan(
			&rel.SourceColumn,
			&rel.TargetTable,
			&rel.TargetColumn,
			&rel.ConstraintName,
		)
		if err != nil {
			return nil, err
		}
		
		rel.SourceTable = tableName
		rel.RelationshipType = "FOREIGN_KEY"
		relationships = append(relationships, &rel)
	}

	return relationships, nil
}

// isJunctionTable determines if a table is a junction table for many-to-many relationships
func (s *SchemaAnalyzerService) isJunctionTable(table *models.TableInfo) bool {
	// Heuristics for junction table detection:
	// 1. Has only foreign key columns (plus optional metadata like timestamps)
	// 2. Primary key is composite of foreign keys
	// 3. Table name suggests relationship (contains underscores, plural forms)
	
	if len(table.Relationships) < 2 {
		return false
	}

	foreignKeyCount := len(table.Relationships)
	totalColumns := len(table.Columns)
	
	// If most columns are foreign keys, likely a junction table
	if float64(foreignKeyCount)/float64(totalColumns) > 0.6 {
		return true
	}

	// Check naming patterns
	junctionPatterns := []string{
		`.*_.*`, // contains underscore
		`.*s_.*s`, // plural_plural pattern
	}
	
	for _, pattern := range junctionPatterns {
		matched, _ := regexp.MatchString(pattern, strings.ToLower(table.Name))
		if matched && foreignKeyCount >= 2 {
			return true
		}
	}

	return false
}

// identifyGraphPatterns identifies common graph database patterns
func (s *SchemaAnalyzerService) identifyGraphPatterns(tables []*models.TableInfo) []*models.GraphPattern {
	var patterns []*models.GraphPattern
	
	// Pattern 1: Star schema (one central table with many relationships)
	starCenters := s.findStarSchemaPatterns(tables)
	for _, center := range starCenters {
		patterns = append(patterns, &models.GraphPattern{
			PatternType: "STAR_SCHEMA",
			CenterTable: center.Name,
			Description: fmt.Sprintf("Star schema with %s as central node", center.Name),
			Confidence:  s.calculatePatternConfidence("STAR_SCHEMA", center),
		})
	}

	// Pattern 2: Hierarchical relationships (self-referencing tables)
	hierarchical := s.findHierarchicalPatterns(tables)
	for _, table := range hierarchical {
		patterns = append(patterns, &models.GraphPattern{
			PatternType: "HIERARCHY",
			CenterTable: table.Name,
			Description: fmt.Sprintf("Hierarchical structure in %s table", table.Name),
			Confidence:  s.calculatePatternConfidence("HIERARCHY", table),
		})
	}

	return patterns
}

// findStarSchemaPatterns identifies tables that are centers of star schemas
func (s *SchemaAnalyzerService) findStarSchemaPatterns(tables []*models.TableInfo) []*models.TableInfo {
	var starCenters []*models.TableInfo
	
	for _, table := range tables {
		// Count incoming relationships (other tables referencing this one)
		incomingRels := 0
		for _, otherTable := range tables {
			for _, rel := range otherTable.Relationships {
				if rel.TargetTable == table.Name {
					incomingRels++
				}
			}
		}
		
		// If many tables reference this one, it's likely a central node
		if incomingRels >= 3 {
			starCenters = append(starCenters, table)
		}
	}
	
	return starCenters
}

// findHierarchicalPatterns identifies tables with self-referencing relationships
func (s *SchemaAnalyzerService) findHierarchicalPatterns(tables []*models.TableInfo) []*models.TableInfo {
	var hierarchical []*models.TableInfo
	
	for _, table := range tables {
		for _, rel := range table.Relationships {
			if rel.TargetTable == table.Name {
				hierarchical = append(hierarchical, table)
				break
			}
		}
	}
	
	return hierarchical
}

// calculatePatternConfidence calculates confidence score for identified patterns
func (s *SchemaAnalyzerService) calculatePatternConfidence(patternType string, table *models.TableInfo) float64 {
	switch patternType {
	case "STAR_SCHEMA":
		// Higher confidence with more relationships
		return min(float64(len(table.Relationships))*0.2, 1.0)
	case "HIERARCHY":
		// High confidence for self-referencing tables
		return 0.9
	default:
		return 0.5
	}
}

// generateTransformationRules creates Neo4j transformation rules based on schema analysis
func (s *SchemaAnalyzerService) generateTransformationRules(result *models.SchemaAnalysisResult) error {
	var rules []*models.TransformationRule
	
	for _, table := range result.Tables {
		if table.GraphType == "NODE" {
			// Generate node creation rule
			nodeRule := s.generateNodeRule(table)
			rules = append(rules, nodeRule)
		} else if table.GraphType == "RELATIONSHIP" {
			// Generate relationship creation rule
			relRule := s.generateRelationshipRule(table)
			rules = append(rules, relRule)
		}
	}
	
	result.GeneratedRules = rules
	return nil
}

// generateNodeRule creates a transformation rule for node creation
func (s *SchemaAnalyzerService) generateNodeRule(table *models.TableInfo) *models.TransformationRule {
	// Generate Neo4j CREATE statement
	cypher := fmt.Sprintf("CREATE (n:%s {", strings.Title(table.Name))
	
	var properties []string
	for _, col := range table.Columns {
		if !s.isForeignKeyColumn(col.Name, table.Relationships) {
			properties = append(properties, fmt.Sprintf("%s: row.%s", col.Name, col.Name))
		}
	}
	
	cypher += strings.Join(properties, ", ") + "})"
	
	return &models.TransformationRule{
		RuleID:      fmt.Sprintf("create_%s_nodes", strings.ToLower(table.Name)),
		RuleType:    "NODE_CREATION",
		SourceTable: table.Name,
		CypherQuery: cypher,
		Description: fmt.Sprintf("Creates %s nodes from %s table", strings.Title(table.Name), table.Name),
		AutoGenerated: true,
		Confidence:  0.8,
	}
}

// generateRelationshipRule creates a transformation rule for relationship creation
func (s *SchemaAnalyzerService) generateRelationshipRule(table *models.TableInfo) *models.TransformationRule {
	if len(table.Relationships) < 2 {
		return nil
	}
	
	// For junction tables, create relationships between the referenced entities
	rel1 := table.Relationships[0]
	rel2 := table.Relationships[1]
	
	relationshipType := s.generateRelationshipType(table.Name, rel1.TargetTable, rel2.TargetTable)
	
	cypher := fmt.Sprintf(
		"MATCH (a:%s {id: row.%s}), (b:%s {id: row.%s}) CREATE (a)-[:%s]->(b)",
		strings.Title(rel1.TargetTable), rel1.SourceColumn,
		strings.Title(rel2.TargetTable), rel2.SourceColumn,
		relationshipType,
	)
	
	return &models.TransformationRule{
		RuleID:      fmt.Sprintf("create_%s_relationships", strings.ToLower(table.Name)),
		RuleType:    "RELATIONSHIP_CREATION",
		SourceTable: table.Name,
		CypherQuery: cypher,
		Description: fmt.Sprintf("Creates %s relationships from %s junction table", relationshipType, table.Name),
		AutoGenerated: true,
		Confidence:  0.7,
	}
}

// isForeignKeyColumn checks if a column is a foreign key
func (s *SchemaAnalyzerService) isForeignKeyColumn(columnName string, relationships []*models.Relationship) bool {
	for _, rel := range relationships {
		if rel.SourceColumn == columnName {
			return true
		}
	}
	return false
}

// generateRelationshipType creates a meaningful relationship type name
func (s *SchemaAnalyzerService) generateRelationshipType(junctionTable, table1, table2 string) string {
	// Remove common suffixes/prefixes and create meaningful relationship name
	cleanJunction := strings.TrimSuffix(strings.ToUpper(junctionTable), "S")
	cleanJunction = strings.ReplaceAll(cleanJunction, "_", "_")
	
	// If junction table name doesn't provide clear relationship name,
	// generate one based on the connected tables
	if !strings.Contains(cleanJunction, strings.ToUpper(table1)) && 
	   !strings.Contains(cleanJunction, strings.ToUpper(table2)) {
		return fmt.Sprintf("%s_TO_%s", strings.ToUpper(table1), strings.ToUpper(table2))
	}
	
	return cleanJunction
}

// min helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
