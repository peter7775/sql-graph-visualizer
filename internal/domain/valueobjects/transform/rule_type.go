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

package transform

type RuleType string

const (
	NodeRule         RuleType = "node"
	RelationshipRule RuleType = "relationship"
)

type NodeMapping struct {
	Type        string `yaml:"type"`
	Key         string `yaml:"key"`
	TargetField string `yaml:"target_field"`
}

type TransformRule struct {
	Name          string            `yaml:"name"`
	SourceTable   string            `yaml:"source_table"`
	SourceSQL     string            `yaml:"source_sql,omitempty"`
	RuleType      RuleType          `yaml:"rule_type"`
	TargetType    string            `yaml:"target_type"`
	Direction     Direction         `yaml:"direction,omitempty"`
	FieldMappings map[string]string `yaml:"field_mappings"`
	RelationType  string            `yaml:"relationship_type,omitempty"`
	SourceNode    *NodeMapping      `yaml:"source_node,omitempty"`
	TargetNode    *NodeMapping      `yaml:"target_node,omitempty"`
	Properties    map[string]string `yaml:"properties,omitempty"`
	Priority      int               `yaml:"priority"`
}

func (rt RuleType) Validate() bool {
	switch rt {
	case NodeRule, RelationshipRule:
		return true
	default:
		return false
	}
}

func ParseDirection(direction string) Direction {
	switch direction {
	case "incoming":
		return Incoming
	case "outgoing":
		return Outgoing
	case "both":
		return Both
	default:
		return Outgoing // Default value
	}
}

// Ensure the Direction constants are defined
const (
	Unknown Direction = iota
	Inbound
	Outbound
)
