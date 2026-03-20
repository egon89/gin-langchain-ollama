#!/usr/bin/env bash

set -e

set -a
source .env
set +a

COMMAND=${1:-up}

shift || true

echo "🚀 Running migration: $COMMAND $@"

docker run --rm \
  -v "$(pwd)/migrations:/migrations" \
  --network host \
  migrate/migrate \
  -path=/migrations \
  -database "$DB_URL" \
  $COMMAND "$@"