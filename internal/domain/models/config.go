/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package models

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

type NodeConfig struct {
	Label      string            `yaml:"label"`
	Properties []PropertyMapping `yaml:"properties"`
}

type PropertyMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

type RelationConfig struct {
	Type string       `yaml:"type"`
	From RelationNode `yaml:"from"`
	To   RelationNode `yaml:"to"`
}

type RelationNode struct {
	Type        string `yaml:"type"`
	Key         string `yaml:"key"`
	TargetField string `yaml:"target_field"`
}

type SourceConfig struct {
	Type        string `yaml:"type"`
	Value       string `yaml:"value"`
	SourceTable string `yaml:"source_table"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

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
	TransformRules []map[string]interface{} `yaml:"transform_rules"`
}
