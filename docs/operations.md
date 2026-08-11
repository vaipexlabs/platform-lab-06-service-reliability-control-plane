# Local Operations Guide

This guide covers the reproducible local reference stack. It is not a
production deployment guide.

## Components

| Component | Address | Purpose |
| --- | --- | --- |
| Checkout demo | `http://localhost:18080` | Emits request and latency telemetry |
| Prometheus | `http://localhost:19090` | Scrapes metrics and evaluates generated rules |
| Grafana | `http://localhost:13000` | Displays provisioned reliability dashboards |

Published ports bind to `127.0.0.1` and are unavailable from other hosts. The
checkout container runs as an unprivileged user with a read-only filesystem,
all Linux capabilities dropped, and `no-new-privileges` enabled.

## Lifecycle

Start and verify the stack:

```bash
./scripts/start-local.sh
docker compose -f deploy/local/compose.yaml ps
curl --fail http://localhost:18080/health/ready
curl --fail http://localhost:19090/-/ready
curl --fail http://localhost:13000/api/health
```

Inspect service logs:

```bash
docker compose -f deploy/local/compose.yaml logs --tail=100 checkout-demo
docker compose -f deploy/local/compose.yaml logs --tail=100 prometheus
docker compose -f deploy/local/compose.yaml logs --tail=100 grafana
```

Inspect generated rules and current alert state:

```bash
sed -n '1,240p' .local/prometheus-rules.yml
curl --fail --silent http://localhost:19090/api/v1/rules
curl --fail --silent http://localhost:19090/api/v1/alerts
```

Stop and remove the project stack:

```bash
./scripts/stop-local.sh
```

The stack uses ephemeral container storage. Cleanup removes local Prometheus
samples and Grafana runtime state; the dashboard and data source are recreated
from version-controlled provisioning files on the next start.

## Common Problems

| Symptom | Check |
| --- | --- |
| Docker connection error | Start Docker Desktop and run `docker info` |
| Port already allocated | Check ports `18080`, `19090`, and `13000`; stop the conflicting local process |
| Dashboard has no data | Generate traffic, confirm the Prometheus target is up, and refresh Grafana |
| Alerts have not fired | Allow for the contract's one-minute rule interval and run `./scripts/verify-alerts.sh` |
| Stale or unexpected metrics | Run `./scripts/stop-local.sh`, then start a clean stack |
| Generated rule error | Validate the contract and inspect `.local/prometheus-rules.yml` plus Prometheus logs |

## Production Adoption Boundary

Before production adoption, provide managed identity, TLS, authorization,
network policy, durable storage, backup and recovery, high availability,
resource limits, audit logging, secret management, retention controls,
upgrade testing, and an owned support model. Replace anonymous Grafana access
and local Docker networking with organization-approved platform services.
