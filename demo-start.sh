#!/bin/bash

# SQL Graph Visualizer - Quick Demo Startup Script
echo "🚀 Starting SQL Graph Visualizer Demo..."

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose not found. Please install Docker Compose."
    exit 1
fi

echo "📦 Pulling required Docker images..."
docker-compose -f docker-compose.demo.yml pull

echo "🏗️  Building application..."
docker-compose -f docker-compose.demo.yml build

echo "🗄️  Starting databases..."
docker-compose -f docker-compose.demo.yml up -d mysql-demo postgres-demo neo4j-demo redis-demo

echo "⏳ Waiting for databases to be ready..."
sleep 30

# Check database health
echo "🔍 Checking database connectivity..."
docker-compose -f docker-compose.demo.yml exec mysql-demo mysqladmin ping -h localhost -u demo_user -pdemopass123 2>/dev/null
if [ $? -eq 0 ]; then
    echo "✅ MySQL is ready"
else
    echo "⚠️  MySQL might still be starting..."
fi

echo "🚀 Starting main application..."
docker-compose -f docker-compose.demo.yml up -d sql-graph-visualizer

echo "🌐 Starting Nginx proxy..."
docker-compose -f docker-compose.demo.yml up -d nginx-demo

echo ""
echo "🎉 Demo is starting up! Please wait about 60 seconds for full initialization..."
echo ""
echo "📱 Access points:"
echo "   🌐 Main Application: http://localhost:3000"
echo "   🔧 GraphQL Playground: http://localhost:8080/graphql"
echo "   📊 Neo4j Browser: http://localhost:7474"
echo "   🔍 API Health Check: http://localhost:8080/api/health"
echo ""
echo "🔐 Demo credentials:"
echo "   Neo4j: neo4j/demopass123"
echo "   MySQL: demo_user/demopass123"
echo "   PostgreSQL: demo_user/demopass123"
echo ""
echo "📋 Useful commands:"
echo "   View logs: docker-compose -f docker-compose.demo.yml logs -f"
echo "   Stop demo: docker-compose -f docker-compose.demo.yml down"
echo "   Restart: docker-compose -f docker-compose.demo.yml restart sql-graph-visualizer"
echo ""

# Monitor application startup
echo "📈 Monitoring application startup..."
for i in {1..30}; do
    if curl -f -s http://localhost:3000/api/health >/dev/null 2>&1; then
        echo "✅ Application is ready!"
        echo "🎯 Open http://localhost:3000 to see the demo!"
        break
    fi
    echo "   ⏳ Still starting... (${i}/30)"
    sleep 2
done

if ! curl -f -s http://localhost:3000/api/health >/dev/null 2>&1; then
    echo "⚠️  Application might still be starting. Check logs:"
    echo "   docker-compose -f docker-compose.demo.yml logs sql-graph-visualizer"
fi
