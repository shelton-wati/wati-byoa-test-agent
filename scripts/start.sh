#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="$ROOT/.env"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

export HTTP_ADDR="${HTTP_ADDR:-:8090}"

mkdir -p bin .wati-byoa-state
go build -o bin/wati-byoa-test-agent ./cmd/wati-byoa-test-agent

echo "Local webhook: http://127.0.0.1:${HTTP_ADDR##*:}/wati/webhook"
exec bin/wati-byoa-test-agent
