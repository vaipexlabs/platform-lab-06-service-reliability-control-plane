#!/usr/bin/env bash

set -euo pipefail

prometheus_url="${PROMETHEUS_URL:-http://localhost:19090}"

for _ in $(seq 1 75); do
  availability_response="$(curl --fail --silent --show-error --get \
    "${prometheus_url}/api/v1/rules" \
    --data-urlencode "type=alert" \
    --data-urlencode "rule_name[]=VaipexRequestAvailabilityFastBurn")"
  latency_response="$(curl --fail --silent --show-error --get \
    "${prometheus_url}/api/v1/rules" \
    --data-urlencode "type=alert" \
    --data-urlencode "rule_name[]=VaipexRequestLatencyFastBurn")"
  if [[ "${availability_response}" == *'"state":"firing","name":"VaipexRequestAvailabilityFastBurn"'* ]] \
    && [[ "${latency_response}" == *'"state":"firing","name":"VaipexRequestLatencyFastBurn"'* ]]; then
    echo "Verified firing fast-burn alerts for availability and latency."
    exit 0
  fi
  sleep 1
done

echo "Expected fast-burn alerts did not fire within 75 seconds." >&2
echo "Availability rule: ${availability_response}" >&2
echo "Latency rule: ${latency_response}" >&2
exit 1
