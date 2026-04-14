#!/bin/sh
set -e


if [ "$INIT_DB" = "1" ]; then
  echo "[entrypoint] INIT_DB=1 -> attempting to initialize MySQL schema"
  if command -v mysql >/dev/null 2>&1; then
    DBCLI="mysql"
  else
    DBCLI="mariadb"
  fi
  if [ -n "$MYSQL_HOST" ] && [ -n "$MYSQL_USER" ] && [ -n "$MYSQL_PASSWORD" ] && [ -n "$MYSQL_DATABASE" ]; then
    echo "[entrypoint] Testing connection to $MYSQL_HOST/$MYSQL_DATABASE"
    if $DBCLI -h "$MYSQL_HOST" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1" "$MYSQL_DATABASE" >/dev/null 2>&1; then
      echo "[entrypoint] MySQL is available, checking tables..."
      TBL_COUNT=$($DBCLI -N -B -h "$MYSQL_HOST" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='${MYSQL_DATABASE}'") || TBL_COUNT=0
      echo "[entrypoint] Table count: $TBL_COUNT"
      if [ "$TBL_COUNT" -lt 5 ] && [ -f "/app/railway-mysql-init.sql" ]; then
        echo "[entrypoint] Loading /app/railway-mysql-init.sql ..."
        $DBCLI -h "$MYSQL_HOST" -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" < /app/railway-mysql-init.sql || echo "[entrypoint] Warning: DB initialization failed"
      fi
    else
      echo "[entrypoint] Warning: MySQL is not available, skipping INIT_DB"
    fi
  else
    echo "[entrypoint] Warning: missing MYSQL_* variables for INIT_DB, skipped"
  fi
fi


exec /app/sql-graph-visualizer
