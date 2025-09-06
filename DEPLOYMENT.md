# Deployment Guide

This guide covers various deployment options for SQL Graph Visualizer, from local development to production cloud deployments.

## 🚀 One-Click Deployments

### Railway (Recommended for Demo)

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/your-template-id)

1. Click the button above
2. Connect your GitHub account
3. Fork the repository
4. Railway will automatically:
   - Deploy the application
   - Set up MySQL database
   - Configure environment variables
   - Provide you with a live URL

**Note:** You'll need to set up Neo4j separately using Neo4j AuraDB (free tier available).

### Heroku

[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/peter7775/sql-graph-visualizer)

1. Click the button above
2. Configure add-ons:
   - **JawsDB MySQL** (Kitefin - Free)
   - **GrapheneDB Neo4j** (Chalk - Free)
3. Set environment variables
4. Deploy and access your live demo

### DigitalOcean App Platform

1. Fork this repository
2. Connect to DigitalOcean App Platform
3. Use the provided `.do/app.yaml` configuration
4. Add Neo4j AuraDB connection details
5. Deploy

## 🐳 Docker Deployment

### Local Demo with Docker Compose

```bash
# Clone the repository
git clone https://github.com/peter7775/sql-graph-visualizer.git
cd sql-graph-visualizer

# Start demo environment
docker-compose -f docker-compose.demo.yml up -d

# Access the application
open http://localhost:3000
```

This will start:
- MySQL with sample e-commerce data
- PostgreSQL with analytics data
- Neo4j graph database
- Redis for caching
- Main application with demo configuration
- Nginx reverse proxy

### Production Docker Setup

```bash
# Build production image
docker build -t sql-graph-visualizer:latest .

# Run with production configuration
docker run -d \
  --name sql-graph-visualizer \
  -p 3000:3000 -p 8080:8080 \
  -e GO_ENV=production \
  -e CONFIG_PATH=/app/config/production.yml \
  sql-graph-visualizer:latest
```

## ☁️ Cloud Deployments

### AWS ECS/EKS

1. **Build and push to ECR:**

```bash
# Build and tag
docker build -t sql-graph-visualizer:latest .
docker tag sql-graph-visualizer:latest YOUR_ECR_URI:latest

# Push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin YOUR_ECR_URI
docker push YOUR_ECR_URI:latest
```

2. **Deploy to ECS:**
   - Use provided `aws/task-definition.json`
   - Configure RDS MySQL and DocumentDB (or self-hosted Neo4j)

3. **Deploy to EKS:**
   - Use Kubernetes manifests in `k8s/` directory
   - Configure persistent volumes for databases

### Google Cloud Run

```bash
# Build and deploy
gcloud builds submit --tag gcr.io/YOUR_PROJECT/sql-graph-visualizer
gcloud run deploy sql-graph-visualizer \
  --image gcr.io/YOUR_PROJECT/sql-graph-visualizer \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --memory 1Gi \
  --cpu 1
```

### Azure Container Instances

```bash
# Create resource group
az group create --name sql-graph-rg --location eastus

# Deploy container
az container create \
  --resource-group sql-graph-rg \
  --name sql-graph-visualizer \
  --image your-registry/sql-graph-visualizer:latest \
  --dns-name-label sql-graph-demo \
  --ports 3000 8080 \
  --environment-variables GO_ENV=demo
```

## 🗄️ Database Setup

### Neo4j Setup Options

#### Option 1: Neo4j AuraDB (Managed Cloud)

1. Go to [Neo4j AuraDB](https://neo4j.com/cloud/aura/)
2. Create free instance
3. Note connection details:
   ```
   NEO4J_URI=neo4j+s://xxx.databases.neo4j.io
   NEO4J_USER=neo4j
   NEO4J_PASSWORD=your-generated-password
   ```

#### Option 2: Self-hosted Neo4j

```bash
# Using Docker
docker run -d \
  --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/yourpassword \
  -e NEO4J_PLUGINS='["apoc"]' \
  neo4j:5.0
```

#### Option 3: Neo4j Desktop

1. Download Neo4j Desktop
2. Create new project and database
3. Install APOC plugin
4. Start database and note connection details

### MySQL/PostgreSQL Setup

#### Managed Options:
- **AWS RDS** - MySQL/PostgreSQL
- **Google Cloud SQL** - MySQL/PostgreSQL  
- **Azure Database** - MySQL/PostgreSQL
- **PlanetScale** - MySQL (serverless)
- **Neon** - PostgreSQL (serverless)

#### Sample Data Setup:

```bash
# MySQL demo data
mysql -h your-host -u user -p < demo/mysql-sample-data.sql

# PostgreSQL demo data
psql -h your-host -U user -d dbname < demo/postgres-sample-data.sql
```

## ⚙️ Environment Configuration

### Required Environment Variables

```bash
# Application Settings
GO_ENV=demo|production
LOG_LEVEL=info
PORT=3000
API_PORT=8080

# Database Connections
MYSQL_HOST=your-mysql-host
MYSQL_PORT=3306
MYSQL_USER=username
MYSQL_PASSWORD=password
MYSQL_DATABASE=dbname

# Neo4j Connection
NEO4J_URI=bolt://your-neo4j-host:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

# Demo Settings
DEMO_MODE=true
AUTO_TRANSFORM=true
ENABLE_BENCHMARKING=false
```

### Configuration Templates

#### Demo Configuration (`config/demo-config.yml`):
- Sample e-commerce data
- Performance benchmarking enabled
- Relaxed security settings
- Auto-transformation on startup

#### Production Configuration (`config/production.yml`):
- Secure connection settings
- Performance optimizations
- Comprehensive logging
- Health checks enabled

## 🔧 Advanced Deployment

### Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -l app=sql-graph-visualizer

# Access via port-forward
kubectl port-forward svc/sql-graph-visualizer 3000:3000
```

### Docker Swarm

```bash
# Initialize swarm
docker swarm init

# Deploy stack
docker stack deploy -c docker-compose.prod.yml sql-graph
```

### Monitoring and Observability

#### Prometheus Metrics

```yaml
# docker-compose.monitoring.yml
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

#### Application Metrics Endpoints:

- `/api/metrics` - Prometheus metrics
- `/api/health` - Health check
- `/api/status` - Application status

## 🔐 Security Considerations

### Production Security Checklist:

- [ ] Enable HTTPS/TLS
- [ ] Configure authentication
- [ ] Set up API rate limiting
- [ ] Use secrets management
- [ ] Enable CORS properly
- [ ] Configure network security groups
- [ ] Set up monitoring and alerting
- [ ] Regular security updates
- [ ] Database connection encryption
- [ ] Input validation enabled

### Secrets Management:

```bash
# Using environment variables
export MYSQL_PASSWORD=$(cat /run/secrets/mysql_password)
export NEO4J_PASSWORD=$(cat /run/secrets/neo4j_password)

# Using HashiCorp Vault
vault kv get -field=password secret/mysql
vault kv get -field=password secret/neo4j
```

## 📊 Performance Optimization

### Production Settings:

```yaml
# Optimized connection pools
mysql:
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h

neo4j:
  max_connection_pool_size: 100
  connection_timeout: 30s

# Processing optimization
processing:
  batch_size: 5000
  parallel_workers: 8
  max_memory_mb: 4096
```

### Scaling Options:

- **Horizontal scaling**: Multiple app instances behind load balancer
- **Vertical scaling**: Increase CPU/memory resources
- **Database scaling**: Read replicas, sharding
- **Caching**: Redis for query results and session data

## 🚨 Troubleshooting

### Common Issues:

1. **Database Connection Timeout**
   ```bash
   # Check connectivity
   nc -zv your-mysql-host 3306
   nc -zv your-neo4j-host 7687
   ```

2. **Out of Memory**
   ```yaml
   processing:
     batch_size: 1000  # Reduce batch size
     max_memory_mb: 2048  # Increase limit
   ```

3. **GraphQL Generation Errors**
   ```bash
   go install github.com/99designs/gqlgen@latest
   gqlgen generate
   ```

4. **Port Conflicts**
   ```bash
   # Check port usage
   lsof -i :3000
   lsof -i :8080
   ```

## 📞 Support

If you encounter issues during deployment:

1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Search [GitHub Issues](https://github.com/peter7775/sql-graph-visualizer/issues)
3. Join [GitHub Discussions](https://github.com/peter7775/sql-graph-visualizer/discussions)
4. Contact: petrstepanek99@gmail.com

---

**Happy Deploying!** 🚀
