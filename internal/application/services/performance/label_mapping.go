package performance

import (
	"regexp"
	"strings"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
)

// labelMappingSource is the Source value of a mapping derived from rules.
const labelMappingSource = "transform_rules"

// fromTableRe finds the first table after FROM in a query.
var fromTableRe = regexp.MustCompile("(?is)\\bfrom\\s+`?([a-z_][a-z0-9_]*)`?")

// ruleSourceTable returns the (lower-case) table a rule reads from, in order of
// preference: the explicit source_table, the value of a "table" source, the
// first FROM table of a query source. It returns "" when none is known.
func ruleSourceTable(src models.SourceConfig) string {
	if t := strings.TrimSpace(src.SourceTable); t != "" {
		return strings.ToLower(t)
	}
	value := strings.TrimSpace(src.Value)
	if strings.EqualFold(src.Type, "table") && value != "" {
		return strings.ToLower(value)
	}
	if m := fromTableRe.FindStringSubmatch(value); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

func isRelationshipRule(r models.TransformationConfig) bool {
	return strings.EqualFold(r.RuleType, "relationship") || r.RelationType != ""
}

// BuildLabelTableMapping derives the mapping between the instance-level Neo4j
// graph and the table-level performance graph from the transformation rules:
// a node rule links its target label to the table it reads, and a
// relationship rule links its relationship type to the edge(s) between the
// tables of its end labels. A relationship read from a third (junction) table
// maps to the two joins through that table.
func BuildLabelTableMapping(rules []models.TransformationConfig) ports.LabelTableMapping {
	mapping := ports.LabelTableMapping{
		Source:        labelMappingSource,
		Labels:        map[string]ports.LabelMapping{},
		Relationships: map[string]ports.RelationshipMapping{},
	}

	// Pass 1: node labels (relationship rules need them to resolve their ends).
	for _, r := range rules {
		table := ruleSourceTable(r.Source)
		if table == "" {
			continue
		}
		if !isRelationshipRule(r) && r.TargetType != "" {
			mapping.Labels[r.TargetType] = ports.LabelMapping{Table: table}
		}
		for _, n := range r.Nodes {
			if n.Label != "" {
				mapping.Labels[n.Label] = ports.LabelMapping{Table: table}
			}
		}
	}

	addRelationship := func(relType, fromLabel, toLabel, junction string) {
		from, okFrom := mapping.Labels[fromLabel]
		to, okTo := mapping.Labels[toLabel]
		if relType == "" || !okFrom || !okTo {
			return
		}
		var edges []string
		if junction == "" || junction == from.Table || junction == to.Table {
			if from.Table == to.Table {
				return
			}
			edges = []string{ports.GraphEdgeID(from.Table, to.Table)}
		} else {
			edges = []string{ports.GraphEdgeID(junction, from.Table), ports.GraphEdgeID(junction, to.Table)}
		}
		mapping.Relationships[relType] = ports.RelationshipMapping{
			Edge: edges[0], Edges: edges, FromLabel: fromLabel, ToLabel: toLabel,
		}
	}

	// Pass 2: relationship types.
	for _, r := range rules {
		junction := ruleSourceTable(r.Source)
		if isRelationshipRule(r) {
			addRelationship(r.RelationType, r.SourceNode.Type, r.TargetNode.Type, junction)
		}
		for _, rel := range r.Relations {
			addRelationship(rel.Type, rel.From.Type, rel.To.Type, junction)
		}
	}

	return mapping
}
