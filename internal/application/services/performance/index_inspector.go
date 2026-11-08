package performance

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"sql-graph-visualizer/internal/application/ports"
)

var indexIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)

type indexInspector struct {
	db     *sql.DB
	dbType string
}

// NewIndexInspector creates a catalog inspector for the selected SQL dialect.
func NewIndexInspector(db *sql.DB, dbType string) (ports.IndexInspector, error) {
	if dbType != "mysql" && dbType != "postgresql" {
		return nil, fmt.Errorf("unsupported index dialect: %s", dbType)
	}
	return &indexInspector{db: db, dbType: dbType}, nil
}

func (i *indexInspector) DatabaseType() string { return i.dbType }

func validateIndexIdentifiers(values ...string) error {
	for _, value := range values {
		if !indexIdentifier.MatchString(value) {
			return fmt.Errorf("invalid SQL identifier: %q", value)
		}
	}
	return nil
}

func (i *indexInspector) IndexExists(ctx context.Context, table, index string) (bool, error) {
	if err := validateIndexIdentifiers(table, index); err != nil {
		return false, err
	}
	if i.db == nil {
		return false, fmt.Errorf("index inspection requires a database")
	}
	query := `SELECT COUNT(*) FROM information_schema.statistics
		 WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`
	if i.dbType == "postgresql" {
		query = `SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2`
	}
	var n int
	err := i.db.QueryRowContext(ctx, query, table, index).Scan(&n)
	return n > 0, err
}

func (i *indexInspector) BuildCreateDDL(spec ports.IndexSpec) (string, error) {
	if err := validateIndexIdentifiers(append([]string{spec.Name, spec.Table}, spec.Columns...)...); err != nil {
		return "", err
	}
	if len(spec.Columns) == 0 {
		return "", fmt.Errorf("index needs at least one column")
	}
	return fmt.Sprintf("CREATE INDEX %s ON %s (%s)", spec.Name, spec.Table, strings.Join(spec.Columns, ", ")), nil
}

func (i *indexInspector) BuildDropDDL(spec ports.IndexSpec) (string, error) {
	if err := validateIndexIdentifiers(append([]string{spec.Name, spec.Table}, spec.Columns...)...); err != nil {
		return "", err
	}
	if i.dbType == "postgresql" {
		return "DROP INDEX IF EXISTS " + spec.Name, nil
	}
	return fmt.Sprintf("DROP INDEX %s ON %s", spec.Name, spec.Table), nil
}
