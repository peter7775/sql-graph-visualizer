/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

import (
	"mysql-graph-visualizer/internal/domain/valueobjects/transform"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformRuleAggregate_ApplyRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     transform.TransformRule
		input    map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "should transform node rule correctly",
			rule: transform.TransformRule{
				Name:       "test_node",
				RuleType:   transform.NodeRule,
				TargetType: "Person",
				FieldMappings: map[string]string{
					"id":    "id",
					"name":  "name",
					"email": "email",
				},
			},
			input: map[string]interface{}{
				"id":    1,
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expected: map[string]interface{}{
				"_type": "Person",
				"id":    1,
				"name":  "John Doe",
				"email": "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "should transform relationship rule correctly",
			rule: transform.TransformRule{
				Name:         "test_relation",
				RuleType:     transform.RelationshipRule,
				RelationType: "WORKS_IN",
				Direction:    transform.DirectionOutgoing,
				SourceNode: &transform.NodeMapping{
					Type:        "Person",
					Key:         "person_id",
					TargetField: "id",
				},
				TargetNode: &transform.NodeMapping{
					Type:        "Department",
					Key:         "dept_id",
					TargetField: "id",
				},
				Properties: map[string]string{
					"since": "start_date",
				},
			},
			input: map[string]interface{}{
				"person_id":  1,
				"dept_id":    2,
				"start_date": "2024-01-24",
			},
			expected: map[string]interface{}{
				"_type":      "WORKS_IN",
				"_direction": transform.DirectionOutgoing,
				"source": map[string]interface{}{
					"type":  "Person",
					"key":   1,
					"field": "id",
				},
				"target": map[string]interface{}{
					"type":  "Department",
					"key":   2,
					"field": "id",
				},
				"properties": map[string]interface{}{
					"since": "2024-01-24",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregate := &TransformRuleAggregate{Rule: tt.rule}
			result, err := aggregate.ApplyRule(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
