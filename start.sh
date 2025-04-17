#!/bin/sh

set -e

/app/wait-for.sh postgres:5432 -- /app/main

echo "start the app"
exec "$@"
