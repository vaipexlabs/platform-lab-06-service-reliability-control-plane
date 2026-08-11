#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

cd "${repository_root}"

echo "Building reliabilityctl"
go build -o "${temporary_directory}/reliabilityctl" ./cmd/reliabilityctl

echo "Validating supported contract"
"${temporary_directory}/reliabilityctl" validate contracts/examples/checkout-service.yaml

echo "Proving invalid contracts are rejected"
for contract_path in testdata/contracts/invalid-*.yaml; do
  if "${temporary_directory}/reliabilityctl" validate "${contract_path}" > /dev/null 2>&1; then
    echo "expected validation failure: ${contract_path}" >&2
    exit 1
  fi
  echo "rejected: ${contract_path}"
done

echo "Contract validation passed."

