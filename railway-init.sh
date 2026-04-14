#!/bin/bash

echo "Initializing Railway MySQL database..."

# Use railway's built-in MySQL connection
echo "Loading demo data into Railway MySQL database..."
railway run -- sh -c "mysql -h \$MYSQL_HOST -u \$MYSQL_USER -p\$MYSQL_PASSWORD \$MYSQL_DATABASE < railway-mysql-init.sql"

if [ $? -eq 0 ]; then
    echo "Database initialized successfully!"
else
    echo "❌ Database initialization failed, but continuing..."
fi

echo "Database setup complete!"
