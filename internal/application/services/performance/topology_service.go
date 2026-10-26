package performance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/sirupsen/logrus"
)

const (
	// topologySourceConfig marks a topology taken from live_demo.topology.
	topologySourceConfig = "config"
	// topologySourceDerived marks a topology derived from queries and schema.
	topologySourceDerived = "queries+schema"

	foreignKeyLookupTimeout = 5 * time.Second
)

// ErrUnknownQuerySet is returned when a topology is requested for a query set
// that is not configured.
var ErrUnknownQuerySet = errors.New("unknown query set")

// foreignKey is one foreign key relationship between two tables.
type foreignKey struct {
	table, column       string
	refTable, refColumn string
}

// TopologyService builds the table-level graph skeleton used by the live
// benchmark UI: tables become nodes and every adjacent pair of tables in a
// query (its JOIN path) or foreign key between used tables becomes an edge.
type TopologyService struct {
	logger    *logrus.Logger
	querySets map[string]ports.CustomBenchmarkConfig
	override  *models.LiveDemoTopology

	// loadForeignKeys reads all foreign keys of the current schema. It is nil
	// when the database type does not support lookup; tests replace it.
	loadForeignKeys func(ctx context.Context) ([]foreignKey, error)
}

// NewTopologyService creates a topology service. db and dbType ("mysql" or
// "postgresql") enable foreign key discovery; either may be empty. override is
// the optional live_demo.topology configuration.
func NewTopologyService(
	logger *logrus.Logger,
	db *sql.DB,
	dbType string,
	querySets map[string]ports.CustomBenchmarkConfig,
	override *models.LiveDemoTopology,
) *TopologyService {
	s := &TopologyService{logger: logger, querySets: querySets, override: override}
	if db != nil {
		switch dbType {
		case "mysql":
			s.loadForeignKeys = func(ctx context.Context) ([]foreignKey, error) {
				return queryForeignKeys(ctx, db, `
					SELECT table_name, column_name, referenced_table_name, referenced_column_name
					FROM information_schema.key_column_usage
					WHERE table_schema = DATABASE() AND referenced_table_name IS NOT NULL`)
			}
		case "postgresql":
			s.loadForeignKeys = func(ctx context.Context) ([]foreignKey, error) {
				return queryForeignKeys(ctx, db, `
					SELECT kcu.table_name, kcu.column_name, ccu.table_name, ccu.column_name
					FROM information_schema.table_constraints tc
					JOIN information_schema.key_column_usage kcu
					  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
					JOIN information_schema.constraint_column_usage ccu
					  ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
					WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = current_schema()`)
			}
		}
	}
	return s
}

func queryForeignKeys(ctx context.Context, db *sql.DB, query string) ([]foreignKey, error) {
	ctx, cancel := context.WithTimeout(ctx, foreignKeyLookupTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []foreignKey
	for rows.Next() {
		var fk foreignKey
		if err := rows.Scan(&fk.table, &fk.column, &fk.refTable, &fk.refColumn); err != nil {
			return nil, err
		}
		out = append(out, fk)
	}
	return out, rows.Err()
}

// Build returns the topology. With an empty querySet the whole configuration
// is used (the live_demo.topology override wins when present); with a query set
// name only that set's tables are included.
func (s *TopologyService) Build(ctx context.Context, querySet string) (*ports.Topology, error) {
	if querySet == "" && s.override != nil && len(s.override.Nodes) > 0 {
		return topologyFromConfig(s.override), nil
	}

	sets := make([]ports.CustomBenchmarkConfig, 0, len(s.querySets))
	if querySet != "" {
		set, ok := s.querySets[querySet]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownQuerySet, querySet)
		}
		sets = append(sets, set)
	} else {
		names := make([]string, 0, len(s.querySets))
		for name := range s.querySets {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sets = append(sets, s.querySets[name])
		}
	}

	nodes := map[string]ports.TopologyNode{}
	edges := map[string]ports.TopologyEdge{}

	for _, set := range sets {
		for _, q := range set.Queries {
			tables := normalizeTables(queryTables(q))
			for _, t := range tables {
				nodes[t] = ports.TopologyNode{ID: t, Table: t, Label: tableLabel(t)}
			}
			for i := 0; i+1 < len(tables); i++ {
				a, b := tables[i], tables[i+1]
				if a == b {
					continue
				}
				id := ports.GraphEdgeID(a, b)
				if _, exists := edges[id]; !exists {
					edges[id] = ports.TopologyEdge{ID: id, Source: a, Target: b, Relation: "joined in queries"}
				}
			}
		}
	}

	if s.loadForeignKeys != nil && len(nodes) > 0 {
		fks, err := s.loadForeignKeys(ctx)
		if err != nil {
			s.logger.WithError(err).Warn("Foreign key discovery failed; topology is derived from queries only")
		}
		for _, fk := range fks {
			child, parent := strings.ToLower(fk.table), strings.ToLower(fk.refTable)
			if _, ok := nodes[child]; !ok {
				continue
			}
			if _, ok := nodes[parent]; !ok || child == parent {
				continue
			}
			id := ports.GraphEdgeID(child, parent)
			edges[id] = ports.TopologyEdge{
				ID: id, Source: child, Target: parent,
				Relation: fmt.Sprintf("%s.%s -> %s.%s", child, strings.ToLower(fk.column), parent, strings.ToLower(fk.refColumn)),
			}
		}
	}

	topology := &ports.Topology{
		Source: topologySourceDerived,
		Nodes:  make([]ports.TopologyNode, 0, len(nodes)),
		Edges:  make([]ports.TopologyEdge, 0, len(edges)),
	}
	for _, n := range nodes {
		topology.Nodes = append(topology.Nodes, n)
	}
	for _, e := range edges {
		topology.Edges = append(topology.Edges, e)
	}
	sort.Slice(topology.Nodes, func(i, j int) bool { return topology.Nodes[i].ID < topology.Nodes[j].ID })
	sort.Slice(topology.Edges, func(i, j int) bool { return topology.Edges[i].ID < topology.Edges[j].ID })
	return topology, nil
}

func topologyFromConfig(cfg *models.LiveDemoTopology) *ports.Topology {
	topology := &ports.Topology{
		Source: topologySourceConfig,
		Nodes:  make([]ports.TopologyNode, 0, len(cfg.Nodes)),
		Edges:  make([]ports.TopologyEdge, 0, len(cfg.Edges)),
	}
	for _, n := range cfg.Nodes {
		table := n.Table
		if table == "" {
			table = n.ID
		}
		label := n.Label
		if label == "" {
			label = n.ID
		}
		topology.Nodes = append(topology.Nodes, ports.TopologyNode{ID: n.ID, Table: table, Label: label})
	}
	for _, e := range cfg.Edges {
		topology.Edges = append(topology.Edges, ports.TopologyEdge{
			ID: ports.GraphEdgeID(e.Source, e.Target), Source: e.Source, Target: e.Target, Relation: e.Relation,
		})
	}
	return topology
}

// QuerySetTables returns the sorted, lower-case tables a query set touches
// (configured `tables` or derived from the SQL).
func QuerySetTables(set ports.CustomBenchmarkConfig) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, q := range set.Queries {
		for _, t := range normalizeTables(queryTables(q)) {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

// normalizeTables lower-cases table names, as used for node IDs.
func normalizeTables(tables []string) []string {
	out := make([]string, 0, len(tables))
	for _, t := range tables {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// tableLabel turns "order_items" into "Order Items".
func tableLabel(table string) string {
	words := strings.FieldsFunc(table, func(r rune) bool { return r == '_' || r == '-' })
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
