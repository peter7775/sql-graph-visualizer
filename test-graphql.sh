#!/bin/bash

# Simple script to test GraphQL functionality

echo "Building the application..."
go build -o mysql-graph-visualizer cmd/main.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Starting the application in the background..."
./mysql-graph-visualizer &
APP_PID=$!

# Wait for the server to start
echo "Waiting for server to start..."
sleep 3

# Test GraphQL config query
echo "Testing GraphQL config query..."
curl -s -X POST http://localhost:8081/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "query { config { neo4j { uri username } } }"}' | jq .

# Test GraphQL graph query (will likely return empty data since no real data is transformed)
echo "Testing GraphQL graph query..."
curl -s -X POST http://localhost:8081/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "query { graph { nodes { id label properties } relationships { from to type properties } } }"}' | jq .

# Test GraphQL mutation
echo "Testing GraphQL transform mutation..."
curl -s -X POST http://localhost:8081/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { transformData }"}' | jq .

# Check if GraphQL playground is accessible
echo "Testing GraphQL playground..."
PLAYGROUND_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/playground)
echo "Playground HTTP status: $PLAYGROUND_STATUS"

# Clean up
echo "Stopping the application..."
kill $APP_PID
wait $APP_PID 2>/dev/null

echo "Cleaning up..."
rm -f mysql-graph-visualizer

echo "GraphQL test completed!"
