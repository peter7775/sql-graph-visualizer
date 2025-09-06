#!/bin/bash
# PostgreSQL initialization script to set up Chinook sample database
# This script runs when PostgreSQL container starts for the first time

set -e

echo "🎵 Starting Chinook database initialization..."

# Download Chinook PostgreSQL schema if not exists
CHINOOK_SQL="/tmp/chinook.sql"

if [ ! -f "$CHINOOK_SQL" ]; then
    echo "📥 Downloading Chinook PostgreSQL schema..."
    curl -L -o "$CHINOOK_SQL" \
        "https://raw.githubusercontent.com/lerocha/chinook-database/master/ChinookDatabase/DataSources/Chinook_PostgreSql.sql" || {
        echo "❌ Failed to download Chinook schema, creating fallback minimal schema..."
        cat > "$CHINOOK_SQL" << 'EOF'
-- Fallback Chinook schema (minimal version)
CREATE TABLE IF NOT EXISTS artist (
    artistid SERIAL PRIMARY KEY,
    name VARCHAR(120)
);

CREATE TABLE IF NOT EXISTS album (
    albumid SERIAL PRIMARY KEY,
    title VARCHAR(160) NOT NULL,
    artistid INTEGER REFERENCES artist(artistid)
);

CREATE TABLE IF NOT EXISTS genre (
    genreid SERIAL PRIMARY KEY,
    name VARCHAR(120)
);

CREATE TABLE IF NOT EXISTS track (
    trackid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    albumid INTEGER REFERENCES album(albumid),
    mediatypeid INTEGER,
    genreid INTEGER REFERENCES genre(genreid),
    composer VARCHAR(220),
    milliseconds INTEGER,
    bytes INTEGER,
    unitprice NUMERIC(10,2)
);

CREATE TABLE IF NOT EXISTS customer (
    customerid SERIAL PRIMARY KEY,
    firstname VARCHAR(40) NOT NULL,
    lastname VARCHAR(20) NOT NULL,
    email VARCHAR(60),
    country VARCHAR(40),
    city VARCHAR(40)
);

-- Insert some sample data
INSERT INTO artist (name) VALUES 
('AC/DC'), ('Accept'), ('Aerosmith'), ('Alanis Morissette'), ('Alice In Chains');

INSERT INTO album (title, artistid) VALUES 
('For Those About To Rock We Salute You', 1),
('Balls to the Wall', 2),
('Restless and Wild', 2),
('Let There Be Rock', 1);

INSERT INTO genre (name) VALUES 
('Rock'), ('Jazz'), ('Metal'), ('Alternative & Punk'), ('Blues');

INSERT INTO track (name, albumid, genreid, composer, milliseconds, unitprice) VALUES 
('For Those About To Rock (We Salute You)', 1, 1, 'Angus Young, Malcolm Young, Brian Johnson', 343719, 0.99),
('Balls to the Wall', 2, 1, NULL, 342562, 0.99),
('Fast As a Shark', 3, 1, 'F. Baltes, S. Kaufman, U. Dirkscneider & W. Hoffman', 230619, 0.99);

INSERT INTO customer (firstname, lastname, email, country, city) VALUES 
('John', 'Smith', 'john.smith@email.com', 'USA', 'New York'),
('Jane', 'Doe', 'jane.doe@email.com', 'Canada', 'Toronto'),
('Bob', 'Johnson', 'bob.johnson@email.com', 'UK', 'London');

EOF
    }
fi

echo "🗄️  Installing Chinook database schema..."

# Connect to the database and execute the schema
export PGPASSWORD="$POSTGRES_PASSWORD"

# Execute the SQL file
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" < "$CHINOOK_SQL"

if [ $? -eq 0 ]; then
    echo "✅ Chinook database installed successfully!"
    
    # Display some statistics
    echo "📊 Database Statistics:"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" << 'EOF'
SELECT 
    schemaname,
    tablename,
    n_tup_ins as "Total Rows"
FROM pg_stat_user_tables 
WHERE n_tup_ins > 0
ORDER BY n_tup_ins DESC;
EOF

    echo ""
    echo "🎵 Chinook database is ready!"
    echo "   📋 Tables: artist, album, track, customer, employee, genre, etc."
    echo "   🔗 Connection: postgresql://postgres:password@localhost:5432/chinook"
    echo "   🌐 pgAdmin: http://localhost:8080 (admin@sqlgraph.local / admin)"
else
    echo "❌ Failed to install Chinook database"
    exit 1
fi

echo "🎉 PostgreSQL initialization completed!"
