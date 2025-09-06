/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"mysql-graph-visualizer/internal/domain/models"

	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
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
	logrus.Debugf("Attempting to load config from: %s", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logrus.Errorf("Config file does not exist: %s", filePath)
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		logrus.Errorf("Error reading config file %s: %v", filePath, err)
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func Load() (*Config, error) {
	// Debug info
	logrus.Debugf("Config loading - GO_ENV: %s, CONFIG_PATH: %s", os.Getenv("GO_ENV"), os.Getenv("CONFIG_PATH"))
	logrus.Debugf("Current working directory: %s", getWorkingDir())
	logrus.Debugf("Project root: %s", findProjectRoot())

	// Check for environment-specific config
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		logrus.Debugf("Using CONFIG_PATH: %s", configPath)
		// If path is not absolute, make it relative to project root
		if !filepath.IsAbs(configPath) {
			configPath = filepath.Join(findProjectRoot(), configPath)
			logrus.Debugf("Resolved to absolute path: %s", configPath)
		}
		return LoadConfig(configPath)
	}

	// Check if we're in test environment
	if os.Getenv("GO_ENV") == "test" {
		configPath := findProjectRoot() + "/config/config-test.yml"
		logrus.Debugf("Using test config: %s", configPath)
		return LoadConfig(configPath)
	}

	// Default config
	configPath := findProjectRoot() + "/config/config.yml"
	logrus.Debugf("Using default config: %s", configPath)
	return LoadConfig(configPath)
}

func getWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return wd
}
func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		// Try to return current directory if we can't get working directory
		logrus.Errorf("Cannot get working directory: %v, using current directory", err)
		return "."
	}

	// First check if go.mod exists in current directory
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd
	}

	// Search parent directories
	originalWd := wd
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	// If we can't find go.mod, return original working directory
	logrus.Warnf("Cannot find project root directory with go.mod, using: %s", originalWd)
	return originalWd
}
