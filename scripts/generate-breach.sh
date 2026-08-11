#!/usr/bin/env bash

set -euo pipefail

REQUESTS="${REQUESTS:-2000}" \
FAILURES="${FAILURES:-400}" \
SLOW_REQUESTS="${SLOW_REQUESTS:-1000}" \
  "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/generate-traffic.sh"

echo "Breach traffic generated: waiting for alert evaluation..."
sleep 3
