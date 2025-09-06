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
