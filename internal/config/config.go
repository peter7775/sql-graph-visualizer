/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"mysql-graph-visualizer/internal/domain/models"

	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Neo4j struct {
		URI      string
		User     string
		Password string
	}
	MySQL struct {
		Host     string
		Port     int
		User     string
		Password string
		Database string
	}
	Server struct {
		Port int
	}
	TransformRules []models.TransformationConfig `yaml:"transform_rules"`
}

func LoadConfig(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}