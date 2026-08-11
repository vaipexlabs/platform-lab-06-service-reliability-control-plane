#!/usr/bin/env bash

set -euo pipefail

requests="${REQUESTS:-2000}"
failures="${FAILURES:-1}"
slow="${SLOW_REQUESTS:-10}"
prometheus_url="${PROMETHEUS_URL:-http://localhost:19090}"
counter_query='sum(vaipex_checkout_requests_total{service="checkout-service"})'

read_request_counter() {
  curl --fail --silent --show-error --get \
    "${prometheus_url}/api/v1/query" \
    --data-urlencode "query=${counter_query}" \
    | sed -n 's/.*"value":\[[^,]*,"\([^"]*\)"\].*/\1/p'
}

baseline=""
for _ in $(seq 1 30); do
  baseline="$(read_request_counter)"
  if [[ -n "${baseline}" ]]; then
    break
  fi
  sleep 1
done
if [[ -z "${baseline}" ]]; then
  echo "Prometheus did not establish a request-counter baseline within 30 seconds." >&2
  exit 1
fi
target="$(awk -v current="${baseline}" -v amount="${requests}" \
  'BEGIN { printf "%.9f", current + amount }')"

curl --fail --silent --show-error \
  "http://localhost:18080/demo/traffic?requests=${requests}&failures=${failures}&slow=${slow}"
echo
echo "Waiting for Prometheus to scrape the completed traffic..."
for _ in $(seq 1 30); do
  observed="$(read_request_counter)"
  observed="${observed:-0}"
  if awk -v current="${observed}" -v expected="${target}" \
    'BEGIN { exit !(current >= expected) }'; then
    echo "Traffic evidence is ready."
    exit 0
  fi
  sleep 1
done

echo "Prometheus did not observe the generated traffic within 30 seconds." >&2
echo "Expected request counter at least ${target}; observed ${observed}." >&2
exit 1
