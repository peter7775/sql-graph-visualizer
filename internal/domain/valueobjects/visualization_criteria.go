package valueobjects

type VisualizationFormat string

const (
	FormatJSON  VisualizationFormat = "json"
	FormatBasic VisualizationFormat = "basic"
)

type VisualizationCriteria struct {
	SearchCriteria
	Format VisualizationFormat
	Limit  int
}

func NewVisualizationCriteria(format VisualizationFormat, limit int) *VisualizationCriteria {
	return &VisualizationCriteria{
		Format: format,
		Limit:  limit,
	}
}
