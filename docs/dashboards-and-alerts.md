# Dashboards, Alerts, and Runbooks

This milestone turns each validated `ServiceReliability` contract into an
operational view and actionable alerts. The contract remains the source of
truth for targets, queries, windows, thresholds, ownership, and evaluation
frequency.

## How It Works

1. `reliabilityctl render-rules` validates the contract and generates a
   Prometheus rule file in `.local/prometheus-rules.yml`.
2. Recording rules calculate good events, total events, attainment, burn rate,
   target ratio, and remaining error budget for every required window.
3. Alert rules compare the short- and long-window burn rates from the same
   objective. Both windows must cross the declared threshold.
4. Grafana reads the recording and alert series from Prometheus through a
   provisioned data source and dashboard.
5. Every alert carries service, objective, team, severity, and runbook labels
   or annotations so responders can understand ownership and take action.

Generated files are local runtime artifacts and are not committed. To inspect
one without starting Docker:

```bash
mkdir -p .local
go run ./cmd/reliabilityctl render-rules \
  --contract contracts/examples/checkout-service.yaml \
  --output .local/prometheus-rules.yml
```

## Alert Strategy

| Signal | Short window | Long window | Threshold | Response |
| --- | --- | --- | ---: | --- |
| Fast burn | 5 minutes | 1 hour | 14.4 | Page and use the [fast-burn runbook](runbooks/fast-burn.md) |
| Slow burn | 30 minutes | 6 hours | 6 | Create follow-up work using the [slow-burn runbook](runbooks/slow-burn.md) |

The paired windows reduce noise: a brief spike does not alert unless sustained
evidence in the companion window also shows accelerated budget consumption.
Thresholds and windows are examples declared in the contract, not hidden in
the generator.

## Healthy Journey

Prerequisites are Docker Desktop, Go 1.26 or newer, and `curl`.

```bash
./scripts/start-local.sh
./scripts/generate-traffic.sh
./scripts/evaluate-live.sh
```

Open the provisioned dashboard:

```text
http://localhost:13000/d/vaipex-service-reliability
```

The default healthy workload produces one failed and ten slow requests across
2,000 requests. Both objectives remain met with a burn rate of `0.5` and 50%
of the error budget remaining.

## Breach Journey

```bash
./scripts/generate-breach.sh
./scripts/verify-alerts.sh
./scripts/evaluate-live.sh
```

The breach workload adds enough failed and slow requests to exceed both
fast-burn thresholds deterministically. Verification can take slightly more
than one minute because the example contract evaluates its rules once per
minute. `verify-alerts.sh` succeeds only when both availability and latency
fast-burn alerts are firing.

The final evaluation reports both objectives as `breached` in structured JSON.
It still exits successfully because this command produces evidence rather than
making a delivery decision. A later decision-interface milestone will convert
that evidence into an explicit proceed, caution, or pause recommendation.

## Components and Ports

| Component | Purpose | Local address |
| --- | --- | --- |
| Checkout demo | Emits request and latency metrics | `http://localhost:18080` |
| Prometheus | Scrapes telemetry and evaluates generated rules | `http://localhost:19090` |
| Grafana | Displays provisioned reliability views | `http://localhost:13000` |

## Cleanup

```bash
./scripts/stop-local.sh
```

Cleanup removes the project containers, network, and volumes. The generated
`.local/prometheus-rules.yml` file remains ignored by Git and will be replaced
the next time the platform starts.
