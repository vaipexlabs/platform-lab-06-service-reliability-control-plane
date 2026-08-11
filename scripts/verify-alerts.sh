#!/usr/bin/env bash

set -euo pipefail

prometheus_url="${PROMETHEUS_URL:-http://localhost:19090}"
query='ALERTS{alertstate="firing",service="checkout-service"}'

for _ in $(seq 1 75); do
  response="$(curl --fail --silent --show-error --get \
    "${prometheus_url}/api/v1/query" \
    --data-urlencode "query=${query}")"
  if [[ "${response}" == *"VaipexRequestAvailabilityFastBurn"* ]] \
    && [[ "${response}" == *"VaipexRequestLatencyFastBurn"* ]]; then
    echo "Verified firing fast-burn alerts for availability and latency."
    exit 0
  fi
  sleep 1
done

echo "Expected fast-burn alerts did not fire within 75 seconds." >&2
echo "${response}" >&2
exit 1
