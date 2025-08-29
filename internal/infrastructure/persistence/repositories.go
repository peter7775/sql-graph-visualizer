/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package infrastructure

import (
	"mysql-graph-visualizer/internal/domain/models"
	"mysql-graph-visualizer/internal/domain/repositories"
	mysqlClient "mysql-graph-visualizer/internal/domain/repositories/mysql"
	neo4jClient "mysql-graph-visualizer/internal/domain/repositories/neo4j"
)

func NewMySQLRepository(config models.Config) (repositories.MySQLRepository, error) {
	client, err := mysqlClient.NewMySQLClient(mysqlClient.MySQLConfig{
		Host:     config.MySQL.Host,
		Port:     config.MySQL.Port,
		User:     config.MySQL.User,
		Password: config.MySQL.Password,
		Database: config.MySQL.Database,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewNeo4jRepository(config models.Config) (repositories.Neo4jRepository, error) {
	client, err := neo4jClient.NewNeo4jClient(neo4jClient.Neo4jConfig{
		URI:      config.Neo4j.URI,
		User:     config.Neo4j.User,
		Password: config.Neo4j.Password,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}
