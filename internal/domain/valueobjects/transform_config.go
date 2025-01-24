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
