package ports

import "context"

// Topology is the table-level graph skeleton shown before and during a live
// benchmark. Node IDs are lower-case table names; edge IDs are produced by
// GraphEdgeID so they match the IDs inside LiveSample graphs.
type Topology struct {
	// Source tells where the topology came from: "config" (live_demo.topology
	// override) or "queries+schema" (query tables plus foreign keys).
	Source string         `json:"source"`
	Nodes  []TopologyNode `json:"nodes"`
	Edges  []TopologyEdge `json:"edges"`
}

// TopologyNode is a table.
type TopologyNode struct {
	ID    string `json:"id"`
	Table string `json:"table"`
	Label string `json:"label"`
}

// TopologyEdge is a relationship (JOIN or foreign key) between two tables.
type TopologyEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// LabelTableMapping maps the instance-level Neo4j graph onto the table-level
// performance graph. It is derived from the transformation rules.
type LabelTableMapping struct {
	// Source is always "transform_rules".
	Source string `json:"source"`
	// Labels maps a Neo4j node label (rule target_type) to its source table.
	Labels map[string]LabelMapping `json:"labels"`
	// Relationships maps a Neo4j relationship type to a table-level edge.
	Relationships map[string]RelationshipMapping `json:"relationships"`
}

// LabelMapping links a Neo4j label to a table.
type LabelMapping struct {
	Table string `json:"table"`
}

// RelationshipMapping links a Neo4j relationship type to an edge ID of the
// performance graph and the labels at its ends.
type RelationshipMapping struct {
	// Edge is the primary edge ID (always set, equals Edges[0]).
	Edge string `json:"edge"`
	// Edges lists every edge the relationship runs through. A relationship
	// stored in a junction table (for example Order CONTAINS_PRODUCT Product
	// through order_items) maps to two edges, one per join. Consumers should
	// aggregate over this list (worst latency, highest load).
	Edges     []string `json:"edges"`
	FromLabel string   `json:"from_label"`
	ToLabel   string   `json:"to_label"`
}

// BenchmarkSampleStorePort persists the live samples of a benchmark run so a
// run can be replayed after the process restarted.
type BenchmarkSampleStorePort interface {
	// AppendSample stores one sample of the run (called once per sample).
	AppendSample(ctx context.Context, benchmarkID string, sample LiveSample) error
	// GetSamples returns the stored samples with Seq greater than afterSeq,
	// ordered by Seq. An unknown run yields an empty slice and no error.
	GetSamples(ctx context.Context, benchmarkID string, afterSeq int) ([]LiveSample, error)
	// HasSamples reports whether any sample is stored for the run.
	HasSamples(ctx context.Context, benchmarkID string) bool
	// DeleteSamples removes the stored samples of a run.
	DeleteSamples(ctx context.Context, benchmarkID string) error
}

// IndexSpec describes one index independent of the SQL dialect.
type IndexSpec struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

// IndexInspector inspects and builds DDL for indexes in a dialect-aware way
// (MySQL and PostgreSQL). Implementations validate every identifier and never
// build anything except plain CREATE INDEX / DROP INDEX statements.
type IndexInspector interface {
	// DatabaseType returns "mysql" or "postgresql".
	DatabaseType() string
	// IndexExists reports whether an index with the given name exists on the
	// table in the current schema.
	IndexExists(ctx context.Context, table, index string) (bool, error)
	// BuildCreateDDL returns the CREATE INDEX statement for the spec.
	BuildCreateDDL(spec IndexSpec) (string, error)
	// BuildDropDDL returns the DROP INDEX statement for the spec
	// (`DROP INDEX name ON table` for MySQL, `DROP INDEX name` for PostgreSQL).
	BuildDropDDL(spec IndexSpec) (string, error)
}

// IndexSuggestion is an index recommendation derived from measured benchmark
// data. Statements and Rollback are executable DDL built by an IndexInspector.
type IndexSuggestion struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Table       string   `json:"table"`
	Columns     []string `json:"columns"`
	Statements  []string `json:"statements"`
	Rollback    []string `json:"rollback"`
	// Queries lists the query patterns that would benefit.
	Queries []string `json:"queries"`
	// Confidence is 0..1.
	Confidence float64 `json:"confidence"`
	// Source is "explain" when confirmed by EXPLAIN, otherwise "heuristic".
	Source string `json:"source"`
	// Applied reports whether an index with the suggested name already exists.
	Applied bool `json:"applied"`
}

// OptimizationState describes the one-click optimization of the live demo as
// shown in GET /api/performance/live/config. Available is false when no
// optimization is configured; CanApply is false outside DEMO_MODE.
type OptimizationState struct {
	Available   bool     `json:"available"`
	CanApply    bool     `json:"can_apply"`
	Name        string   `json:"name,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Applied     bool     `json:"applied"`
	Statements  []string `json:"statements,omitempty"`
}

// IndexAdvisorPort suggests indexes for a finished benchmark run. queries are
// the definitions of the query set the run executed (they carry the parameters
// needed to EXPLAIN a statement).
type IndexAdvisorPort interface {
	SuggestIndexes(ctx context.Context, result *BenchmarkResult, queries []CustomQueryDefinition) ([]IndexSuggestion, error)
}
