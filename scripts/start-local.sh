#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${repository_root}/deploy/local/compose.yaml"
local_directory="${repository_root}/.local"

command -v docker > /dev/null 2>&1 || { echo "Required command not found: docker" >&2; exit 1; }
docker compose version > /dev/null

cd "${repository_root}"
mkdir -p "${local_directory}"
go run ./cmd/reliabilityctl render-rules \
  --contract contracts/examples/checkout-service.yaml \
  --output "${local_directory}/prometheus-rules.yml"

docker compose -f "${compose_file}" up --build --detach

for _ in $(seq 1 30); do
  if curl --fail --silent http://localhost:18080/health/ready > /dev/null \
    && curl --fail --silent http://localhost:19090/-/ready > /dev/null \
    && curl --fail --silent http://localhost:13000/api/health > /dev/null; then
    sleep 2
    echo "Local reliability platform is ready."
    echo "Grafana: http://localhost:13000/d/vaipex-service-reliability"
    exit 0
  fi
  sleep 1
done

echo "Local platform did not become ready within 30 seconds." >&2
docker compose -f "${compose_file}" ps >&2
exit 1
