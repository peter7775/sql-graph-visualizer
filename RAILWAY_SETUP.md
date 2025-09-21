# Railway Deployment Setup Guide

This guide explains how to configure your Railway deployment with proper environment variables for MySQL and Neo4j Aura connections.

## Required Environment Variables

### Database Configuration (MySQL on Railway)

Set these variables in your Railway project settings:

```bash
# MySQL Database Connection
MYSQLHOST=your-mysql-host.railway.internal
MYSQLPORT=3306
MYSQLUSER=your-mysql-username
MYSQLPASSWORD=your-mysql-password
MYSQL_DATABASE=your-database-name
```

### Neo4j Aura Configuration

Set these variables for your Neo4j Aura instance:

```bash
# Neo4j Aura Connection
NEO4J_URI=bolt+s://your-aura-instance.databases.neo4j.io:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-aura-password
```

### Additional Configuration

```bash
# Railway Configuration
PORT=8080
RAILWAY_ENVIRONMENT=production

# Application Configuration
LOG_LEVEL=info
GO_ENV=production
```

## How to Set Environment Variables in Railway

1. Go to your Railway project dashboard
2. Click on your service
3. Navigate to the "Variables" tab
4. Add each environment variable with the corresponding value

## Database Setup Requirements

### MySQL Database Schema

Your MySQL database should contain the Sakila sample database with these tables:
- `actor` (actor_id, first_name, last_name)
- `film` (film_id, title, description, release_year, rating, length)  
- `category` (category_id, name)
- `film_actor` (actor_id, film_id)
- `film_category` (film_id, category_id)

You can download and install the Sakila sample database from:
https://dev.mysql.com/doc/sakila/en/

### Neo4j Aura Setup

1. Create a free Neo4j Aura instance at https://aura.neo4j.io/
2. Note down the connection details (URI, username, password)
3. The application will automatically create the graph nodes and relationships from your MySQL data

## Application Behavior

1. **Configuration Loading**: The app loads `config/cloud-config.yml` when `RAILWAY_ENVIRONMENT` is set
2. **Environment Override**: Database and Neo4j connection details from environment variables override config file defaults
3. **Data Transformation**: MySQL data is transformed into Neo4j graph according to the rules in the config file
4. **Health Check**: Available at `/api/health` endpoint for Railway deployment monitoring

## Troubleshooting

### Common Issues:

1. **"dial tcp: lookup demo.host"**: MySQL environment variables are not set properly
2. **"Neo4j connection failed"**: Check Neo4j Aura credentials and URI format
3. **"Health check failed"**: Database connections are not working

### Debug Information:

The application logs detailed connection information at startup, including:
- Configuration file used
- Environment variables detected
- Database connection status
- Neo4j connection status

## Deployment Commands

The Railway deployment uses:
- **Build**: Docker build using `Dockerfile`
- **Start**: `./sql-graph-visualizer` 
- **Health Check**: `GET /api/health` (300s timeout)
- **Ports**: 3000 (web UI), 8080 (API/health)

## Example Environment Variables File

Create a `.env` file for local testing (not committed to git):

```bash
MYSQLHOST=localhost
MYSQLPORT=3306
MYSQLUSER=root
MYSQLPASSWORD=your_password
MYSQL_DATABASE=sakila

NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_neo4j_password

PORT=8080
LOG_LEVEL=debug
```