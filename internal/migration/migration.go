package migration

import (
	"database/sql"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

func migrateTable(mysqlDB *sql.DB, neo4jDriver neo4j.Driver, migration MigrationConfig) error {
	rows, err := fetchData(mysqlDB, migration.SourceTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		// Implementace převodu řádku z MySQL do Neo4j uzlu
	}

	return nil
}