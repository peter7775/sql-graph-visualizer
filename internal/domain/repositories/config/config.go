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

// Package config provides configuration loading from YAML files.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sql-graph-visualizer/internal/domain/models"
	"strings"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

// configRoot is the only directory from which application configuration files
// may be loaded. Configuration paths are always resolved relative to it.
const configRoot = "config"

// Load loads the application configuration from YAML files.
func Load() (*models.Config, error) {
	configFile, err := resolveConfigFile()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Loading configuration from YAML file: %s", configFile)

	data, err := readConfigFile(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logrus.Errorf("Configuration file does not exist: %s", configFile)
		} else {
			logrus.Errorf("Error reading file: %v", err)
		}
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		logrus.Errorf("Error parsing YAML: %v", err)
		return nil, fmt.Errorf("invalid configuration file %s: %w", configFile, err)
	}

	logrus.Info("Configuration loaded successfully:")
	logrus.Infof("- MySQL: %s:%d/%s", config.MySQL.Host, config.MySQL.Port, config.MySQL.Database)
	logrus.Infof("- Neo4j: %s", config.Neo4j.URI)
	logrus.Infof("- Transform rules count: %d", len(config.TransformRules))

	for i, rule := range config.TransformRules {
		logrus.Infof("  Rule %d: %s (%s) -> %s", i+1, rule.Name, rule.RuleType, rule.TargetType)
		logrus.Infof("    Field mappings: %+v", rule.FieldMappings)
	}

	return &config, nil
}

// resolveConfigFile returns a configuration file path relative to configRoot.
func resolveConfigFile() (string, error) {
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		logrus.Infof("Using CONFIG_PATH environment variable: %s", envPath)

		// CONFIG_PATH is intentionally limited to a path inside the project's
		// config directory. Absolute paths would permit reading arbitrary
		// files from the host filesystem.
		if filepath.IsAbs(envPath) {
			return "", fmt.Errorf("CONFIG_PATH must be relative to %s: %s", configRoot, envPath)
		}

		return safeConfigPath(envPath)
	}

	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		logrus.Info("Railway environment detected, using cloud-config.yml")
		return safeConfigPath("cloud-config.yml")
	}

	return safeConfigPath("config.yml")
}

// safeConfigPath validates a configuration file path and normalizes it for
// use with os.Root.
func safeConfigPath(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("config path must not be empty")
	}

	cleanPath := filepath.Clean(filePath)

	// IsLocal rejects absolute paths, empty paths, traversal outside the
	// current directory, and Windows reserved device names.
	if !filepath.IsLocal(cleanPath) {
		return "", fmt.Errorf("invalid config path: %s", filePath)
	}

	relativePath := filepath.ToSlash(cleanPath)

	// Defensive check for traversal components before handing the path to
	// os.Root.
	if strings.Contains(relativePath, "..") {
		return "", fmt.Errorf("invalid config path: %s", filePath)
	}

	return relativePath, nil
}

// readConfigFile reads a YAML file only from the project's config directory.
func readConfigFile(relativePath string) ([]byte, error) {
	root, err := os.OpenRoot(configRoot)
	if err != nil {
		return nil, fmt.Errorf("open config root %q: %w", configRoot, err)
	}
	defer root.Close()

	if _, err := root.Stat(relativePath); err != nil {
		return nil, err
	}

	return root.ReadFile(relativePath)
}

func findProjectRoot() string {
	// In Railway/production environment, assume we're in /app.
	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		// Check if we're in Docker container with /app directory.
		if _, err := os.Stat("/app"); err == nil {
			logrus.Info("Railway deployment detected, using /app as project root")
			return "/app"
		}

		if wd, err := os.Getwd(); err == nil {
			logrus.Infof("Using current working directory as project root: %s", wd)
			return wd
		}
	}

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
			// Last resort for Railway: try /app.
			if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
				logrus.Warn("Cannot find go.mod, using /app as fallback in Railway")
				return "/app"
			}

			// Do NOT fatal in production containers; fall back to current dir.
			logrus.Error("Cannot find project root directory; falling back to current working directory")
			return wd
		}

		wd = parent
	}
}
