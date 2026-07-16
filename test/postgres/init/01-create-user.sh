#!/bin/bash
set -e

# The application role must NOT be a superuser — superusers bypass Row-Level
# Security, defeating tenant isolation. It runs the migrations on startup, so it
# needs CREATE on the schema; it owns the tables it creates and thus has full DML.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER scorecard WITH PASSWORD 'scorecard_password';
    GRANT CONNECT ON DATABASE scorecard TO scorecard;
    GRANT USAGE, CREATE ON SCHEMA public TO scorecard;
EOSQL

echo "Application role 'scorecard' created successfully"
