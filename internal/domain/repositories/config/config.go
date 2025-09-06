/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */


package config

import (
	"fmt"
	"mysql-graph-visualizer/internal/domain/models"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

func Load() (*models.Config, error) {
	configPath := findProjectRoot() + "/config/config.yml"
	logrus.Infof("Loading configuration from YAML file: %s", configPath)

	// Validate path to prevent directory traversal
	cleanPath := filepath.Clean(configPath)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid config path: %s", configPath)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		logrus.Errorf("Error reading file: %v", err)
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		logrus.Errorf("Error parsing YAML: %v", err)
		return nil, err
	}

	logrus.Infof("Configuration loaded successfully:")
	logrus.Infof("- MySQL: %s:%d/%s", config.MySQL.Host, config.MySQL.Port, config.MySQL.Database)
	logrus.Infof("- Neo4j: %s", config.Neo4j.URI)
	logrus.Infof("- Transform rules count: %d", len(config.TransformRules))

	for i, rule := range config.TransformRules {
		logrus.Infof("  Rule %d: %s (%s) -> %s", i+1, rule.Name, rule.RuleType, rule.TargetType)
		logrus.Infof("    Field mappings: %+v", rule.FieldMappings)
	}

	return &config, nil
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Cannot get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			logrus.Fatalf("Cannot find project root directory")
			return ""
		}
		wd = parent
	}
}
