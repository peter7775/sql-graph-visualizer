/*
 * MySQL Graph Visualizer - Direct Database Service Tests
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package services

import (
	"testing"

	"mysql-graph-visualizer/internal/domain/models"
)

// TestDirectDatabaseService_ValidateConfiguration tests configuration validation
func TestDirectDatabaseService_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		config    *models.MySQLConfig
		expectErr bool
	}{
		{
			name: "Valid configuration",
		config: &models.MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				User:     "testuser",
				Username: "testuser",
				Database: "testdb",
				Security: models.SecurityConfig{
					ConnectionTimeout: 30,
					QueryTimeout:      300,
					MaxConnections:    5,
				},
			},
			expectErr: false,
		},
		{
			name:      "Nil configuration",
			config:    nil,
			expectErr: true,
		},
		{
			name: "Missing host",
			config: &models.MySQLConfig{
				Port:     3306,
				User:     "testuser",
				Database: "testdb",
			},
			expectErr: true,
		},
		{
			name: "Missing database",
			config: &models.MySQLConfig{
				Host: "localhost",
				Port: 3306,
				User: "testuser",
			},
			expectErr: true,
		},
		{
			name: "Missing user",
			config: &models.MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &DirectDatabaseService{
				config: tt.config,
			}
			
			err := service.ValidateConfiguration()
			
			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestDirectDatabaseService_GetConfiguration tests configuration retrieval
func TestDirectDatabaseService_GetConfiguration(t *testing.T) {
	config := &models.MySQLConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "testuser",
		Database: "testdb",
	}

	service := &DirectDatabaseService{
		config: config,
	}

	result := service.GetConfiguration()

	if result != config {
		t.Errorf("Expected configuration to match, got different instance")
	}
}
