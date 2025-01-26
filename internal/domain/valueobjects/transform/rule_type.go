/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
	case "inbound":
		return Inbound
	case "outbound":
		return Outbound
	default:
		return Unknown
	}
}

// Ensure the Direction constants are defined
const (
	Unknown Direction = iota
	Inbound
	Outbound
)
