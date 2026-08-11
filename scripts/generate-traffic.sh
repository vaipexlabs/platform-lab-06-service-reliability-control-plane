#!/usr/bin/env bash

set -euo pipefail

requests="${REQUESTS:-2000}"
failures="${FAILURES:-1}"
slow="${SLOW_REQUESTS:-10}"

curl --fail --silent --show-error \
  "http://localhost:18080/demo/traffic?requests=${requests}&failures=${failures}&slow=${slow}"
echo
echo "Waiting for Prometheus to scrape the completed traffic..."
sleep 2
echo "Traffic evidence is ready."
