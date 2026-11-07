package performance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

const indexLatencyThreshold = 20 * time.Millisecond

// IndexAdvisor recommends indexes only for measured, slow configured queries.
type IndexAdvisor struct {
	logger    *logrus.Logger
	db        *sql.DB
	inspector ports.IndexInspector
}

// NewIndexAdvisor constructs an advisor. It never executes the measured query.
func NewIndexAdvisor(logger *logrus.Logger, db *sql.DB, inspector ports.IndexInspector) *IndexAdvisor {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &IndexAdvisor{logger: logger, db: db, inspector: inspector}
}

type indexCandidate struct {
	table, column string
	equality      bool
}

var sqlTokens = regexp.MustCompile(`(?s)/\*.*?\*/|--[^\r\n]*|#[^\r\n]*|'(?:''|[^'])*'|\$[0-9]+|\?|[A-Za-z_][A-Za-z0-9_]*|[0-9]+(?:\.[0-9]+)?|<=|>=|<>|!=|[^\s]`)

// candidateColumns deliberately supports only simple, unquoted, single-scope
// SQL. Ambiguous names and complex expressions are skipped rather than guessed.
func candidateColumns(query string) []indexCandidate {
	tokens := sqlTokens.FindAllString(query, -1)
	clean := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if strings.HasPrefix(token, "/*") || strings.HasPrefix(token, "--") || strings.HasPrefix(token, "#") {
			continue
		}
		clean = append(clean, token)
	}
	if len(clean) == 0 || (!strings.EqualFold(clean[0], "SELECT") && !strings.EqualFold(clean[0], "UPDATE")) {
		return nil
	}
	aliases := map[string]string{}
	tables := map[string]bool{}
	keyword := func(s string) bool {
		return strings.Contains(" WHERE JOIN LEFT RIGHT INNER OUTER FULL CROSS ON SET ORDER GROUP LIMIT HAVING USING OFFSET ", " "+strings.ToUpper(s)+" ")
	}
	for n, token := range clean {
		if token == ";" && n != len(clean)-1 {
			return nil
		}
		if token == "`" || token == `"` || token == "'" || token == "\\" || token == "/" || token == "$" {
			return nil
		}
		if strings.HasPrefix(token, "'") && strings.Contains(token, "\\") {
			return nil
		}
		switch strings.ToUpper(token) {
		case "UNION", "WITH", "EXISTS", "OR", "USING":
			return nil
		case "SELECT":
			if n != 0 {
				return nil
			}
		case "FROM", "JOIN", "UPDATE":
			if n+1 >= len(clean) || !indexIdentifier.MatchString(clean[n+1]) {
				return nil
			}
			table := clean[n+1]
			if n+2 < len(clean) && (clean[n+2] == "." || clean[n+2] == ",") {
				return nil
			}
			aliases[table], tables[table] = table, true
			a := n + 2
			if a < len(clean) && strings.EqualFold(clean[a], "AS") {
				a++
			}
			if a < len(clean) && indexIdentifier.MatchString(clean[a]) && !keyword(clean[a]) {
				if old, exists := aliases[clean[a]]; exists && old != table {
					return nil
				}
				aliases[clean[a]] = table
				a++
			}
			if a < len(clean) && clean[a] == "," {
				return nil
			}
		}
	}
	resolve := func(pos int) (indexCandidate, int, bool) {
		if pos >= len(clean) || !indexIdentifier.MatchString(clean[pos]) {
			return indexCandidate{}, pos, false
		}
		col, table, end := clean[pos], "", pos+1
		if end+1 < len(clean) && clean[end] == "." {
			table, col, end = aliases[col], clean[end+1], end+2
		} else if len(tables) == 1 {
			for t := range tables {
				table = t
			}
		}
		return indexCandidate{table: table, column: col}, end, table != "" && indexIdentifier.MatchString(col)
	}
	out := []indexCandidate{}
	predicate := false
	for n := 0; n < len(clean); n++ {
		word := strings.ToUpper(clean[n])
		if word == "WHERE" || word == "ON" {
			predicate = true
			continue
		}
		if word == "GROUP" || word == "ORDER" || word == "LIMIT" || word == "SET" {
			predicate = false
		}
		if !predicate || (n > 0 && (clean[n-1] == "." || clean[n-1] == "(")) {
			continue
		}
		candidate, end, ok := resolve(n)
		if !ok || end+1 >= len(clean) {
			continue
		}
		op := strings.ToUpper(clean[end])
		switch op {
		case "=", "<", ">", "<=", ">=", "IN", "BETWEEN", "LIKE":
		default:
			continue
		}
		rhs := clean[end+1]
		if op == "LIKE" && (!strings.HasPrefix(rhs, "'") || !strings.HasSuffix(rhs, "%'") || strings.Contains(rhs[1:len(rhs)-2], "%") || strings.Contains(rhs, "_")) {
			continue
		}
		candidate.equality = op == "=" || op == "IN"
		if other, _, joined := resolve(end + 1); joined && op == "=" {
			other.equality = true
			out = append(out, other)
		} else if rhs != "?" && !strings.HasPrefix(rhs, "$") && !strings.HasPrefix(rhs, "'") && rhs != "(" && (rhs[0] < '0' || rhs[0] > '9') {
			continue
		}
		out = append(out, candidate)
		n = end
	}
	return out
}

type tableRecommendation struct {
	equal, ranges []string
	queries       []string
	impact        float64
	confirmed     bool
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// SuggestIndexes derives recommendations from measured latency and execution
// counts, excluding existing leading index columns and primary keys.
func (a *IndexAdvisor) SuggestIndexes(ctx context.Context, result *ports.BenchmarkResult, definitions []ports.CustomQueryDefinition) ([]ports.IndexSuggestion, error) {
	out := []ports.IndexSuggestion{}
	if result == nil || a.inspector == nil || a.db == nil {
		return out, nil
	}
	defs := map[string]ports.CustomQueryDefinition{}
	for _, definition := range definitions {
		defs[definition.Query] = definition
	}
	recommendations := map[string]*tableRecommendation{}
	for _, measured := range result.QueryResults {
		if measured.AverageTime <= indexLatencyThreshold || (!strings.EqualFold(measured.QueryType, "SELECT") && !strings.EqualFold(measured.QueryType, "UPDATE")) {
			continue
		}
		definition, ok := defs[measured.QueryPattern]
		if !ok {
			continue
		}
		candidates := candidateColumns(definition.Query)
		if len(candidates) == 0 {
			continue
		}
		scans, explained := a.explain(ctx, definition)
		for _, candidate := range candidates {
			indexed, err := a.leadingColumnIndexed(ctx, candidate.table, candidate.column)
			if err != nil {
				return nil, fmt.Errorf("inspect leading index columns: %w", err)
			}
			if indexed || (explained && !scans[candidate.table]) {
				continue
			}
			rec := recommendations[candidate.table]
			if rec == nil {
				rec = &tableRecommendation{}
				recommendations[candidate.table] = rec
			}
			if candidate.equality {
				rec.equal = appendUnique(rec.equal, candidate.column)
			} else {
				rec.ranges = appendUnique(rec.ranges, candidate.column)
			}
			if len(rec.queries) == 0 || rec.queries[len(rec.queries)-1] != definition.Query {
				rec.queries = appendUnique(rec.queries, definition.Query)
				rec.impact += float64(measured.AverageTime) * float64(measured.ExecutionCount)
			}
			rec.confirmed = rec.confirmed || scans[candidate.table]
		}
	}
	tables := make([]string, 0, len(recommendations))
	for table := range recommendations {
		tables = append(tables, table)
	}
	sort.Slice(tables, func(i, j int) bool {
		x, y := recommendations[tables[i]].impact, recommendations[tables[j]].impact
		if x == y {
			return tables[i] < tables[j]
		}
		return x > y
	})
	for _, table := range tables {
		rec := recommendations[table]
		columns := append([]string{}, rec.equal...)
		for _, column := range rec.ranges {
			columns = appendUnique(columns, column)
		}
		if len(columns) > 3 {
			columns = columns[:3]
		}
		name := "idx_" + table + "_" + strings.Join(columns, "_")
		if len(name) > 63 {
			name = name[:63]
		}
		spec := ports.IndexSpec{Name: name, Table: table, Columns: columns}
		create, err := a.inspector.BuildCreateDDL(spec)
		if err != nil {
			return nil, err
		}
		drop, err := a.inspector.BuildDropDDL(spec)
		if err != nil {
			return nil, err
		}
		applied, err := a.inspector.IndexExists(ctx, table, name)
		if err != nil {
			return nil, err
		}
		source, confidence := "heuristic", 0.55
		if rec.confirmed {
			source, confidence = "explain", 0.9
		}
		out = append(out, ports.IndexSuggestion{
			ID: name, Title: "Index " + table + " (" + strings.Join(columns, ", ") + ")",
			Description: "Measured slow predicates can benefit from a leading-column index; validate with the production workload.",
			Table:       table, Columns: columns, Statements: []string{create}, Rollback: []string{drop},
			Queries: rec.queries, Confidence: confidence, Source: source, Applied: applied,
		})
	}
	return out, nil
}

func (a *IndexAdvisor) leadingColumnIndexed(ctx context.Context, table, column string) (bool, error) {
	if err := validateIndexIdentifiers(table, column); err != nil {
		return false, err
	}
	query := `SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE()
		AND table_name = ? AND column_name = ? AND seq_in_index = 1`
	if a.inspector.DatabaseType() == "postgresql" {
		query = `SELECT COUNT(*) FROM pg_index i JOIN pg_class t ON t.oid = i.indrelid
			JOIN pg_namespace n ON n.oid = t.relnamespace
			JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = i.indkey[0]
			WHERE n.nspname = current_schema() AND t.relname = $1 AND a.attname = $2 AND i.indisvalid
			AND i.indpred IS NULL`
	}
	var count int
	err := a.db.QueryRowContext(ctx, query, table, column).Scan(&count)
	return count > 0, err
}

func (a *IndexAdvisor) explain(ctx context.Context, definition ports.CustomQueryDefinition) (map[string]bool, bool) {
	scans := map[string]bool{}
	prefix := "EXPLAIN "
	if a.inspector.DatabaseType() == "postgresql" {
		prefix = "EXPLAIN (FORMAT JSON) "
	}
	rows, err := a.db.QueryContext(ctx, prefix+definition.Query, definition.Parameters...)
	if err != nil && a.inspector.DatabaseType() == "postgresql" {
		if literal, ok := explainLiterals(definition); ok {
			rows, err = a.db.QueryContext(ctx, prefix+literal)
		}
	}
	if err != nil {
		a.logger.WithError(err).Debug("EXPLAIN unavailable; using conservative index heuristic")
		return scans, false
	}
	defer func() { _ = rows.Close() }()
	if a.inspector.DatabaseType() == "postgresql" {
		var raw []byte
		if !rows.Next() || rows.Scan(&raw) != nil {
			return scans, false
		}
		var plan any
		if json.Unmarshal(raw, &plan) != nil {
			return scans, false
		}
		collectSequentialScans(plan, scans)
		return scans, rows.Err() == nil
	}
	columns, err := rows.Columns()
	if err != nil {
		return scans, false
	}
	for rows.Next() {
		values := make([]sql.RawBytes, len(columns))
		pointers := make([]any, len(columns))
		for n := range values {
			pointers[n] = &values[n]
		}
		if rows.Scan(pointers...) != nil {
			return scans, false
		}
		table, access, extra := "", "", ""
		for n, column := range columns {
			switch column {
			case "table":
				table = string(values[n])
			case "type":
				access = string(values[n])
			case "Extra":
				extra = string(values[n])
			}
		}
		// A full "index" scan (for example a primary key walked backwards for
		// ORDER BY ... LIMIT while filtering on another column) reads the table
		// just like ALL unless it is a covering index-only scan.
		if mysqlScansTable(access, extra) {
			scans[table] = true
			// MySQL reports aliases; resolve them via parsed query candidates.
			for _, candidate := range candidateColumns(definition.Query) {
				alias := regexp.MustCompile(`(?i)\b(?:FROM|JOIN|UPDATE)\s+` + regexp.QuoteMeta(candidate.table) + `\s+(?:AS\s+)?` + regexp.QuoteMeta(table) + `\b`)
				if alias.MatchString(definition.Query) {
					scans[candidate.table] = true
				}
			}
		}
	}
	return scans, rows.Err() == nil
}

// mysqlScansTable reports whether a MySQL EXPLAIN row describes a scan that
// reads the table rather than seeking through an index.
func mysqlScansTable(access, extra string) bool {
	return access == "ALL" || (access == "index" && !strings.Contains(extra, "Using index"))
}

func collectSequentialScans(value any, scans map[string]bool) {
	switch node := value.(type) {
	case map[string]any:
		if node["Node Type"] == "Seq Scan" {
			if table, ok := node["Relation Name"].(string); ok {
				scans[table] = true
			}
		}
		for _, child := range node {
			collectSequentialScans(child, scans)
		}
	case []any:
		for _, child := range node {
			collectSequentialScans(child, scans)
		}
	}
}

func explainLiterals(definition ports.CustomQueryDefinition) (string, bool) {
	var out strings.Builder
	last, next := 0, 0
	for _, bounds := range sqlTokens.FindAllStringIndex(definition.Query, -1) {
		token := definition.Query[bounds[0]:bounds[1]]
		if token != "?" && !strings.HasPrefix(token, "$") {
			continue
		}
		position := next
		if token == "?" {
			next++
		} else {
			n, err := strconv.Atoi(token[1:])
			if err != nil {
				return "", false
			}
			position = n - 1
		}
		if position < 0 || position >= len(definition.Parameters) {
			return "", false
		}
		literal, ok := explainLiteral(definition.Parameters[position])
		if !ok {
			return "", false
		}
		out.WriteString(definition.Query[last:bounds[0]])
		out.WriteString(literal)
		last = bounds[1]
	}
	out.WriteString(definition.Query[last:])
	return out.String(), true
}

func explainLiteral(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "NULL", true
	case string:
		if strings.ContainsAny(v, "\\\x00") {
			return "", false
		}
		return "'" + strings.ReplaceAll(v, "'", "''") + "'", true
	case bool:
		return strconv.FormatBool(v), true
	case int, int32, int64, uint, uint32, uint64, float32, float64:
		text := fmt.Sprint(v)
		return text, regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`).MatchString(text)
	default:
		return "", false
	}
}
