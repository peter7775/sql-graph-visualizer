/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"mysql-graph-visualizer/internal/application/ports"
	"os"

	"mysql-graph-visualizer/internal/domain/aggregates/transform"
	transformvo "mysql-graph-visualizer/internal/domain/valueobjects/transform"

	"gopkg.in/yaml.v3"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) ports.MySQLPort {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) FetchData() ([]map[string]interface{}, error) {
	// Načtení konfigurace
	config := getMySQLConfig()

	// Příprava výsledků
	var results []map[string]interface{}

	// Načtení dat z tabulek definovaných v transform_rules
	for _, ruleConfig := range config.transform_rules {
		rule := transform.TransformRuleAggregate{
			Rule: transformvo.TransformRule{
				SourceTable: ruleConfig.SourceTable,
				RuleType:    transformvo.RuleType(ruleConfig.RuleType),
				// Další pole podle potřeby
			},
		}

		query := fmt.Sprintf("SELECT * FROM %s", rule.Rule.SourceTable)
		rows, err := r.db.Query(query)
		if err != nil {
			return nil, fmt.Errorf("error querying table %s: %v", rule.Rule.SourceTable, err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("error getting columns for table %s: %v", rule.Rule.SourceTable, err)
		}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, fmt.Errorf("error scanning row from table %s: %v", rule.Rule.SourceTable, err)
			}

			rowMap := make(map[string]interface{})
			for i, col := range columns {
				rowMap[col] = values[i]
			}

			// Přidání informace o zdrojové tabulce
			rowMap["_table"] = rule.Rule.SourceTable

			// Aplikace transformace
			transformedData, err := rule.ApplyRule(rowMap)
			if err != nil {
				return nil, fmt.Errorf("error applying transformation for table %s: %v", rule.Rule.SourceTable, err)
			}

			results = append(results, transformedData.(map[string]interface{}))
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating rows for table %s: %v", rule.Rule.SourceTable, err)
		}
	}

	return results, nil
}

func (r *MySQLRepository) Close() error {
	return r.db.Close()
}

func getMySQLConfig() *mysqlConfig {
	// Načtení konfigurace ze souboru
	configData, err := os.ReadFile("config/config.yml")
	if err != nil {
		log.Fatalf("Nelze načíst konfigurační soubor: %v", err)
	}

	// Parsování YAML
	var config struct {
		TransformRules []struct {
			SourceTable string `yaml:"source_table"`
			RuleType    string `yaml:"rule_type"`
		} `yaml:"transform_rules"`
	}

	if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Nelze parsovat konfigurační soubor: %v", err)
	}

	log.Printf("Načtené transform_rules: %+v", config.TransformRules)

	return &mysqlConfig{
		transform_rules: config.TransformRules,
	}
}

// Definice struktury mysqlConfig
type mysqlConfig struct {
	transform_rules []struct {
		SourceTable string `yaml:"source_table"`
		RuleType    string `yaml:"rule_type"`
	}
}

// Přidání funkce applyTransformation
func applyTransformation(rule struct{ SourceTable string `yaml:"source_table"` }, data map[string]interface{}) (map[string]interface{}, error) {
	// Zde by měla být logika pro aplikaci transformace
	// Prozatím vrátíme data beze změny
	return data, nil
}
