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

// Package config provides application configuration loading and validation.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sql-graph-visualizer/internal/domain/models"
	"strings"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// configRoot is the only directory from which application configuration files
// may be loaded. All configuration paths are resolved relative to this root.
const configRoot = "config"

// Config represents the main application configuration structure.
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

// LoadConfig loads configuration from a file located inside the project's
// config directory. The filePath must be relative to the config directory.
func LoadConfig(filePath string) (*Config, error) {
	logrus.Debugf("Attempting to load config from: %s", filePath)

	relativePath, err := safeConfigPath(filePath)
	if err != nil {
		return nil, err
	}

	data, err := readConfigFile(relativePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logrus.Errorf("Config file does not exist: %s", relativePath)
		} else {
			logrus.Errorf("Error reading config file %s: %v", relativePath, err)
		}
		return nil, err
	}

	var config Config
	// #nosec G709
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid configuration file %s: %w", relativePath, err)
	}

	return &config, nil
}

// safeConfigPath validates an externally supplied configuration path and
// converts it into a path relative to configRoot.
func safeConfigPath(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("config path must not be empty")
	}

	cleanPath := filepath.Clean(filePath)

	// Reject absolute paths, traversal outside the configuration root, and
	// Windows drive/root-style paths.
	if filepath.IsAbs(cleanPath) || !filepath.IsLocal(cleanPath) {
		return "", fmt.Errorf("invalid config path: %s", filePath)
	}

	// Normalize separators for os.Root and reject any remaining traversal
	// component defensively.
	relativePath := filepath.ToSlash(cleanPath)
	if strings.Contains(relativePath, "..") {
		return "", fmt.Errorf("invalid config path: %s", filePath)
	}

	return relativePath, nil
}

// readConfigFile reads a file only from configRoot. os.Root prevents access
// outside that directory, including traversal through ".." or symlinks.
func readConfigFile(relativePath string) ([]byte, error) {
	root, err := os.OpenRoot(configRoot)
	if err != nil {
		return nil, fmt.Errorf("open config root %q: %w", configRoot, err)
	}
	defer closeRoot(root)

	if _, err := root.Stat(relativePath); err != nil {
		return nil, err
	}

	return root.ReadFile(relativePath)
}

// closeRoot closes an os.Root and logs a close failure instead of silently
// discarding its returned error.
func closeRoot(root *os.Root) {
	if err := root.Close(); err != nil {
		logrus.Errorf("Error closing config root: %v", err)
	}
}

// Load loads configuration using default logic and environment variables.
func Load() (*Config, error) {
	// Debug info
	logrus.Debugf(
		"Config loading - GO_ENV: %s, CONFIG_PATH: %s",
		os.Getenv("GO_ENV"),
		os.Getenv("CONFIG_PATH"),
	)
	logrus.Debugf("Current working directory: %s", getWorkingDir())
	logrus.Debugf("Project root: %s", findProjectRoot())

	// Check for environment-specific config.
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		logrus.Debugf("Using CONFIG_PATH: %s", configPath)

		// CONFIG_PATH is interpreted relative to the project config directory.
		// Absolute paths are intentionally rejected to prevent loading files
		// from arbitrary locations.
		if filepath.IsAbs(configPath) {
			return nil, fmt.Errorf("CONFIG_PATH must be relative to %s: %s", configRoot, configPath)
		}

		return LoadConfig(configPath)
	}

	// Check if we're in test environment.
	if os.Getenv("GO_ENV") == "test" {
		logrus.Debug("Using test config: config/config-test.yml")
		return LoadConfig("config-test.yml")
	}

	logrus.Debug("Using default config: config/config.yml")
	return LoadConfig("config.yml")
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
		// Try to return current directory if we can't get working directory.
		logrus.Errorf("Cannot get working directory: %v, using current directory", err)
		return "."
	}

	// First check if go.mod exists in current directory.
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd
	}

	// Search parent directories.
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

	logrus.Warnf("Cannot find project root directory with go.mod, using: %s", originalWd)
	return originalWd
}
