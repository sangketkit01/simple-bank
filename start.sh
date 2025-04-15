#!/bin/sh

set -e

echo "run db migration"
export $(grep -v '^#' /app/app.env | xargs)
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
exec "$@"
