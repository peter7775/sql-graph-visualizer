# Railway Deployment Instructions

## Problem Summary
Your Railway deployment is failing because the application cannot connect to the database. The error `dial tcp: lookup demo.host` indicates that MySQL environment variables are not properly set.

## Solution Steps

### 1. Set Environment Variables in Railway

Go to your Railway project dashboard → Your service → Variables tab, and add these environment variables:

#### MySQL Database (Required)
```bash
MYSQLHOST=your-mysql-host.railway.internal
MYSQLPORT=3306
MYSQLUSER=your-mysql-username  
MYSQLPASSWORD=your-mysql-password
MYSQL_DATABASE=your-database-name
```

#### Neo4j Aura (Required)
```bash
NEO4J_URI=bolt+s://your-instance.databases.neo4j.io:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-aura-password
```

#### Application Settings (Optional)
```bash
PORT=8080
RAILWAY_ENVIRONMENT=production
LOG_LEVEL=info
```

### 2. Database Requirements

Your MySQL database needs the **Sakila sample database**. It should contain these tables:
- `actor` (actor_id, first_name, last_name)
- `film` (film_id, title, description, release_year, rating, length)
- `category` (category_id, name)  
- `film_actor` (actor_id, film_id)
- `film_category` (film_id, category_id)

If you don't have Sakila data:
1. Download from: https://dev.mysql.com/doc/sakila/en/
2. Or use Railway's MySQL template with Sakila pre-installed

### 3. Get Your Database Connection Details

#### For Railway MySQL:
1. Go to Railway Dashboard
2. Find your MySQL service  
3. Go to "Connect" tab
4. Copy the connection details (host, user, password, database)

#### For Neo4j Aura:
1. Go to https://aura.neo4j.io/
2. Create a free instance (if you haven't)
3. Copy the connection URI, username, password

### 4. Verify Configuration

After setting environment variables, redeploy your Railway service. The application will:

1. Load `config/cloud-config.yml` (because `RAILWAY_ENVIRONMENT` is set)
2. Override database settings with your environment variables  
3. Connect to MySQL and transform data to Neo4j
4. Start the web server on the PORT specified

### 5. Test the Deployment

Your deployment should show these logs on success:
```
✅ Configuration loaded successfully
✅ MySQL connection successful  
✅ Neo4j connection successful
✅ Data transformation successful
✅ Server started
```

Health check endpoint: `https://your-app.railway.app/api/health`

### 6. Troubleshooting

#### Common Issues:

**"dial tcp: lookup demo.host"**
- Environment variables are not set properly
- Check variable names (use `MYSQLHOST` not `MYSQL_HOST`)

**"Neo4j connection failed"**  
- Check Neo4j URI format: `bolt+s://...` for Aura
- Verify username/password

**"Health check timeout"**
- Database connection is taking too long
- Check if Sakila data exists in your MySQL database

#### Debug Information:

The app logs all environment variables (without passwords) at startup to help debugging.

### 7. Expected Result

Once working, your app will:
- Transform MySQL Sakila data into a Neo4j graph
- Show actors, films, categories as nodes
- Show relationships (Actor→Film, Film→Category) as edges  
- Provide visualization at your Railway URL
- GraphQL API at `/graphql`
- Health check at `/api/health`

## Example Environment Variables

Here's what your Railway environment variables should look like:

```bash
# MySQL (replace with your actual values)
MYSQLHOST=mysql.railway.internal
MYSQLPORT=3306  
MYSQLUSER=root
MYSQLPASSWORD=your_mysql_password
MYSQL_DATABASE=sakila

# Neo4j (replace with your actual values)
NEO4J_URI=bolt+s://abcd1234.databases.neo4j.io:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_neo4j_password

# App settings
PORT=8080
RAILWAY_ENVIRONMENT=production
LOG_LEVEL=info
```

After setting these variables, redeploy your Railway service and the health check should pass! 🚀