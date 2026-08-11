# Live Prometheus Evidence

The live journey demonstrates the adapter boundary between provider-specific
telemetry and provider-neutral reliability evaluation.

## Runtime Components

| Component | Responsibility |
| --- | --- |
| Checkout demo | Exposes deterministic request counters and a duration histogram |
| Prometheus | Scrapes the service every second and executes contract query templates |
| `reliabilityctl collect` | Replaces `$window`, executes instant queries, and writes a `ReliabilityEvidence` snapshot |
| Evaluation engine | Applies the existing SLO and error-budget calculations without knowing Prometheus |

Prometheus is pinned by image digest. The checkout demo is built as a static,
non-root container. Host ports `18080` and `19090` avoid common development
ports while container traffic continues to use the standard service ports.

## Query Window Contract

PromQL fields contain a required `$window` placeholder:

```promql
sum(increase(vaipex_checkout_requests_total{service="checkout-service"}[$window]))
```

The collector replaces `$window` with the objective's declared `30d` window,
parses the rendered expression with the Prometheus parser, and submits it to
`/api/v1/query` at one evaluation timestamp. This prevents a five-minute query
from being presented as 30-day SLO evidence.

Each query must aggregate to zero or one result series. An empty result becomes
zero traffic; multiple series are rejected with guidance to aggregate the
query. HTTP errors, Prometheus API errors, malformed samples, and timeouts stop
collection rather than producing misleading evidence.

## Local Journey

Prerequisites are Docker Desktop, Go 1.26 or newer, and `curl`.

```bash
./scripts/start-local.sh
./scripts/generate-traffic.sh
./scripts/evaluate-live.sh
./scripts/stop-local.sh
```

The default traffic contains 2,000 requests, one failure, and ten requests
slower than 500 milliseconds. It deterministically produces 99.95%
availability and 99.5% latency attainment. Both objectives consume 50% of
their error budgets.

Override traffic characteristics to explore another state:

```bash
FAILURES=4 SLOW_REQUESTS=40 ./scripts/generate-traffic.sh
./scripts/evaluate-live.sh
```

## Manual Collection

Write live evidence to a chosen file:

```bash
go run ./cmd/reliabilityctl collect \
  --contract contracts/examples/checkout-service.yaml \
  --prometheus-url http://localhost:19090 \
  --output /tmp/checkout-evidence.json
```

Then evaluate it independently:

```bash
go run ./cmd/reliabilityctl evaluate \
  --contract contracts/examples/checkout-service.yaml \
  --evidence /tmp/checkout-evidence.json
```

`collect --at <RFC3339>` is available for repeatable adapter tests. Normal
local use omits it so the collector evaluates the current instant.

