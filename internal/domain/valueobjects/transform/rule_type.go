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
}

func (rt RuleType) Validate() bool {
	switch rt {
	case NodeRule, RelationshipRule:
		return true
	default:
		return false
	}
}
