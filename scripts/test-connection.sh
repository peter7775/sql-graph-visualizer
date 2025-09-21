#!/bin/bash

# Test Database Connection Script for Railway Deployment
# This script helps debug database connection issues

echo "=== SQL Graph Visualizer - Connection Test ==="
echo

# Display environment variables (without sensitive data)
echo "Environment Variables:"
echo "- RAILWAY_ENVIRONMENT: ${RAILWAY_ENVIRONMENT:-NOT_SET}"
echo "- PORT: ${PORT:-NOT_SET}"
echo "- LOG_LEVEL: ${LOG_LEVEL:-NOT_SET}"
echo

echo "MySQL Environment Variables:"
echo "- MYSQLHOST: ${MYSQLHOST:-NOT_SET}"
echo "- MYSQL_HOST: ${MYSQL_HOST:-NOT_SET}"
echo "- MYSQLUSER: ${MYSQLUSER:-NOT_SET}"
echo "- MYSQL_USER: ${MYSQL_USER:-NOT_SET}"
echo "- MYSQL_DATABASE: ${MYSQL_DATABASE:-NOT_SET}"
echo "- MYSQLPORT: ${MYSQLPORT:-NOT_SET}"
echo "- MYSQL_PORT: ${MYSQL_PORT:-NOT_SET}"
echo "- MYSQLPASSWORD: $([ -n "$MYSQLPASSWORD" ] && echo "SET" || echo "NOT_SET")"
echo "- MYSQL_PASSWORD: $([ -n "$MYSQL_PASSWORD" ] && echo "SET" || echo "NOT_SET")"
echo

echo "Neo4j Environment Variables:"
echo "- NEO4J_URI: $([ -n "$NEO4J_URI" ] && echo "SET" || echo "NOT_SET")"
echo "- NEO4J_USER: ${NEO4J_USER:-NOT_SET}"
echo "- NEO4J_PASSWORD: $([ -n "$NEO4J_PASSWORD" ] && echo "SET" || echo "NOT_SET")"
echo

# Test MySQL connection if mysql client is available
if command -v mysql >/dev/null 2>&1; then
    echo "Testing MySQL connection..."
    
    # Determine which env vars to use
    MYSQL_HOST_VAR="${MYSQLHOST:-$MYSQL_HOST}"
    MYSQL_USER_VAR="${MYSQLUSER:-$MYSQL_USER}"
    MYSQL_PASS_VAR="${MYSQLPASSWORD:-$MYSQL_PASSWORD}"
    MYSQL_PORT_VAR="${MYSQLPORT:-${MYSQL_PORT:-3306}}"
    
    if [ -n "$MYSQL_HOST_VAR" ] && [ -n "$MYSQL_USER_VAR" ] && [ -n "$MYSQL_PASS_VAR" ]; then
        echo "Attempting connection to: $MYSQL_USER_VAR@$MYSQL_HOST_VAR:$MYSQL_PORT_VAR/$MYSQL_DATABASE"
        
        mysql -h"$MYSQL_HOST_VAR" -P"$MYSQL_PORT_VAR" -u"$MYSQL_USER_VAR" -p"$MYSQL_PASS_VAR" -e "SELECT 'MySQL connection successful!' AS status;" "$MYSQL_DATABASE" 2>/dev/null
        if [ $? -eq 0 ]; then
            echo "✅ MySQL connection: SUCCESS"
            
            # Test for Sakila database structure
            echo
            echo "Testing Sakila database structure..."
            mysql -h"$MYSQL_HOST_VAR" -P"$MYSQL_PORT_VAR" -u"$MYSQL_USER_VAR" -p"$MYSQL_PASS_VAR" -e "SHOW TABLES LIKE 'actor';" "$MYSQL_DATABASE" 2>/dev/null | grep -q "actor"
            if [ $? -eq 0 ]; then
                echo "✅ Sakila 'actor' table: FOUND"
            else
                echo "❌ Sakila 'actor' table: NOT FOUND"
            fi
            
            mysql -h"$MYSQL_HOST_VAR" -P"$MYSQL_PORT_VAR" -u"$MYSQL_USER_VAR" -p"$MYSQL_PASS_VAR" -e "SHOW TABLES LIKE 'film';" "$MYSQL_DATABASE" 2>/dev/null | grep -q "film"
            if [ $? -eq 0 ]; then
                echo "✅ Sakila 'film' table: FOUND"
            else
                echo "❌ Sakila 'film' table: NOT FOUND"
            fi
            
            # Count records
            echo
            echo "Sample data counts:"
            mysql -h"$MYSQL_HOST_VAR" -P"$MYSQL_PORT_VAR" -u"$MYSQL_USER_VAR" -p"$MYSQL_PASS_VAR" -e "SELECT COUNT(*) as actor_count FROM actor;" "$MYSQL_DATABASE" 2>/dev/null
            mysql -h"$MYSQL_HOST_VAR" -P"$MYSQL_PORT_VAR" -u"$MYSQL_USER_VAR" -p"$MYSQL_PASS_VAR" -e "SELECT COUNT(*) as film_count FROM film;" "$MYSQL_DATABASE" 2>/dev/null
        else
            echo "❌ MySQL connection: FAILED"
        fi
    else
        echo "❌ MySQL connection: Missing required environment variables"
    fi
else
    echo "⚠️  MySQL client not available for testing"
fi

echo
echo "=== Test Complete ==="

# If this is running in Railway, also test the application health endpoint
if [ -n "$RAILWAY_ENVIRONMENT" ]; then
    echo
    echo "Testing application health endpoint..."
    sleep 2  # Give the app a moment to start
    curl -f http://localhost:${PORT:-8080}/api/health 2>/dev/null
    if [ $? -eq 0 ]; then
        echo
        echo "✅ Application health check: SUCCESS"
    else
        echo
        echo "❌ Application health check: FAILED"
    fi
fi