/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package valueobjects

type TransformConfig struct {
	SourceType      string
	TargetType      string
	Priority        int
	TransformFields map[string]string
}

func NewTransformConfig(sourceType, targetType string, priority int) TransformConfig {
	return TransformConfig{
		SourceType:      sourceType,
		TargetType:      targetType,
		Priority:        priority,
		TransformFields: make(map[string]string),
	}
}
