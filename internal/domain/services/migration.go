package migration

import (
	"database/sql"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type MigrationConfig struct {
	SourceTable string
	Neo4jDriver neo4j.Driver
	// Add other fields as needed
}

func fetchData(mysqlDB *sql.DB, tableName string) (*sql.Rows, error) {
	query := "SELECT * FROM " + tableName
	return mysqlDB.Query(query)
}

func migrateTable(mysqlDB *sql.DB, migration MigrationConfig) error {
	rows, err := fetchData(mysqlDB, migration.SourceTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	session := migration.Neo4jDriver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	for rows.Next() {
		// Implementace převodu řádku z MySQL do Neo4j uzlu
	}

	return nil
}
