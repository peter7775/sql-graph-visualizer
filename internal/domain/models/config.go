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


package models

// TransformationConfig represents a single transformation rule configuration.
type TransformationConfig struct {
	Name          string            `yaml:"name"`
	Source        SourceConfig      `yaml:"source"`
	Nodes         []NodeConfig      `yaml:"nodes"`
	Relations     []RelationConfig  `yaml:"relations"`
	FieldMappings map[string]string `yaml:"field_mappings"`
	SourceNode    RelationNode      `yaml:"source_node"`
	TargetNode    RelationNode      `yaml:"target_node"`
	RelationType  string            `yaml:"relationship_type,omitempty"`
	TargetType    string            `yaml:"target_type,omitempty"`
	RuleType      string            `yaml:"rule_type,omitempty"`
	Direction     string            `yaml:"direction,omitempty"`
	Properties    map[string]string `yaml:"properties,omitempty"`
	Priority      int               `yaml:"priority,omitempty"`
}

// NodeConfig represents node configuration for transformation rules.
type NodeConfig struct {
	Label      string            `yaml:"label"`
	Properties []PropertyMapping `yaml:"properties"`
}

// PropertyMapping represents mapping between source and target properties.
type PropertyMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// RelationConfig represents relationship configuration between nodes.
type RelationConfig struct {
	Type string       `yaml:"type"`
	From RelationNode `yaml:"from"`
	To   RelationNode `yaml:"to"`
}

// RelationNode represents a node in a relationship configuration.
type RelationNode struct {
	Type        string `yaml:"type"`
	Key         string `yaml:"key"`
	TargetField string `yaml:"target_field"`
}

// SourceConfig represents data source configuration for transformations.
type SourceConfig struct {
	Type        string `yaml:"type"`
	Value       string `yaml:"value"`
	SourceTable string `yaml:"source_table"`
}

// MySQLConfig represents MySQL database connection configuration.
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// Neo4jConfig represents Neo4j database connection configuration.
type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Config represents the main application configuration.
type Config struct {
	MySQL struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`
	Neo4j struct {
		URI      string `yaml:"uri"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"neo4j"`
	TransformRules []TransformationConfig `yaml:"transform_rules"`
}
