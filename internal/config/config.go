/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

type Config struct {
	MySQL struct {
		Host     string
		Port     int
		User     string
		Password string
		Database string
	}
	Neo4j struct {
		URI      string
		Username string
		Password string
	}
	Server struct {
		Port int
	}
}

func Load() (*Config, error) {
	cfg := &Config{}
	cfg.MySQL.Host = "mysql-test"
	cfg.MySQL.Port = 3306
	cfg.MySQL.User = "root"
	cfg.MySQL.Password = "testpass"
	cfg.MySQL.Database = "testdb"

	cfg.Neo4j.URI = "bolt://neo4j-test:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "testpass"

	cfg.Server.Port = 8080

	return cfg, nil
}
