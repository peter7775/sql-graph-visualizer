/*
 * SQL Graph Visualizer - Integration Tests for MySQL Performance Schema collection
 *
 * Copyright (c) 2025
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 */

package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/services/performance"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceSchemaIntegrationTestSuite exercises PerformanceSchemaAdapter
// against the same sakila MySQL test database used by
// DirectDatabaseIntegrationTestSuite (docker-compose service on 127.0.0.1:3308).
type PerformanceSchemaIntegrationTestSuite struct {
	suite.Suite
	db      *sql.DB
	adapter *performance.PerformanceSchemaAdapter
	ctx     context.Context
}

func (suite *PerformanceSchemaIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	if os.Getenv("INTEGRATION_TESTS") != "true" {
		suite.T().Skip("Integration tests skipped - set INTEGRATION_TESTS=true to enable")
	}

	dsn := "sakila_user:sakila123@tcp(127.0.0.1:3308)/sakila"
	db, err := sql.Open("mysql", dsn)
	require.NoError(suite.T(), err, "should open MySQL connection")

	ctx, cancel := context.WithTimeout(suite.ctx, 5*time.Second)
	defer cancel()
	require.NoError(suite.T(), db.PingContext(ctx), "test database should be reachable")

	suite.db = db
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	suite.adapter = performance.NewPerformanceSchemaAdapter(db, logger, nil)
}

func (suite *PerformanceSchemaIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		_ = suite.db.Close()
	}
}

func (suite *PerformanceSchemaIntegrationTestSuite) TestCollectPerformanceData() {
	data, err := suite.adapter.CollectPerformanceData(suite.ctx)
	require.NoError(suite.T(), err, "CollectPerformanceData should succeed against a live MySQL instance")
	require.NotNil(suite.T(), data)
	suite.T().Logf("collected %d statement stats, %d table IO stats", len(data.StatementStats), len(data.TableIOStats))

	metrics := suite.adapter.ConvertToPerformanceMetrics(data)
	require.NotNil(suite.T(), metrics)

	queryPerf := suite.adapter.ConvertToQueryPerformance(data)
	suite.T().Logf("converted %d query performance entries", len(queryPerf))
}

func TestPerformanceSchemaIntegration(t *testing.T) {
	suite.Run(t, new(PerformanceSchemaIntegrationTestSuite))
}
