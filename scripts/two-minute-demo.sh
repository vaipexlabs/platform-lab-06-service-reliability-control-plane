#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
compose_file="${repository_root}/deploy/local/compose.yaml"

cleanup() {
  docker compose -f "${compose_file}" down --remove-orphans > /dev/null 2>&1 || true
  rm -rf "${temporary_directory}"
}
trap cleanup EXIT

for required_command in docker go curl; do
  command -v "${required_command}" > /dev/null 2>&1 || {
    echo "Required command not found: ${required_command}" >&2
    exit 1
  }
done
docker compose version > /dev/null

cd "${repository_root}"

echo "Vaipex Service Reliability Control Plane"
echo "========================================="
echo
echo "1/5 Validate the reliability contract"
go build -o "${temporary_directory}/reliabilityctl" ./cmd/reliabilityctl
"${temporary_directory}/reliabilityctl" validate contracts/examples/checkout-service.yaml

echo
echo "2/5 Start the local reliability platform"
docker compose -f "${compose_file}" down --remove-orphans > /dev/null 2>&1 || true
./scripts/start-local.sh

echo
echo "3/5 Prove healthy evidence recommends proceed"
./scripts/generate-traffic.sh
"${temporary_directory}/reliabilityctl" collect \
  --contract contracts/examples/checkout-service.yaml \
  --prometheus-url http://localhost:19090 \
  --output "${temporary_directory}/healthy-evidence.json"
"${temporary_directory}/reliabilityctl" evaluate \
  --contract contracts/examples/checkout-service.yaml \
  --evidence "${temporary_directory}/healthy-evidence.json" \
  > "${temporary_directory}/healthy-evaluation.json"
"${temporary_directory}/reliabilityctl" decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence "${temporary_directory}/healthy-evidence.json" \
  > "${temporary_directory}/healthy-decision.json"
cat "${temporary_directory}/healthy-evaluation.json"
cat "${temporary_directory}/healthy-decision.json"
grep --quiet '"recommendation": "proceed"' "${temporary_directory}/healthy-decision.json"

echo
echo "4/5 Prove a reliability breach fires alerts"
./scripts/generate-breach.sh
./scripts/verify-alerts.sh

echo
echo "5/5 Prove breached evidence recommends pause"
"${temporary_directory}/reliabilityctl" collect \
  --contract contracts/examples/checkout-service.yaml \
  --prometheus-url http://localhost:19090 \
  --output "${temporary_directory}/breach-evidence.json"
"${temporary_directory}/reliabilityctl" evaluate \
  --contract contracts/examples/checkout-service.yaml \
  --evidence "${temporary_directory}/breach-evidence.json" \
  > "${temporary_directory}/breach-evaluation.json"
"${temporary_directory}/reliabilityctl" decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence "${temporary_directory}/breach-evidence.json" \
  --active-severity page \
  > "${temporary_directory}/breach-decision.json"
cat "${temporary_directory}/breach-evaluation.json"
cat "${temporary_directory}/breach-decision.json"
grep --quiet '"recommendation": "pause"' "${temporary_directory}/breach-decision.json"

echo
echo "Demo passed: healthy evidence recommended proceed; breached evidence and active alerts recommended pause."
