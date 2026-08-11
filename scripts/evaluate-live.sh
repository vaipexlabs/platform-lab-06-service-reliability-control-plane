#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

cd "${repository_root}"
go build -o "${temporary_directory}/reliabilityctl" ./cmd/reliabilityctl

"${temporary_directory}/reliabilityctl" collect \
  --contract contracts/examples/checkout-service.yaml \
  --prometheus-url http://localhost:19090 \
  --output "${temporary_directory}/evidence.json"

"${temporary_directory}/reliabilityctl" evaluate \
  --contract contracts/examples/checkout-service.yaml \
  --evidence "${temporary_directory}/evidence.json"
