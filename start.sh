#!/bin/sh

set -e

echo "run db migration"
export $(grep -v '^#' /app/app.env | xargs)
echo "DB_SOURCE: $DB_SOURCE"
echo "app.env content:"
cat /app/app.env
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
exec "$@"
