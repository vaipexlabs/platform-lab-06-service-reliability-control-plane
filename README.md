# Vaipex Service Reliability Control Plane

An open reference implementation for turning service-level indicators into
clear reliability objectives, actionable error budgets, and explainable
delivery decisions.

Developed by **Vaipex Labs** for the developer and platform engineering
community.

![Status](https://img.shields.io/badge/Status-Reference%20Complete-6D5DFB)
![Reliability](https://img.shields.io/badge/Focus-SLOs%20%26%20Error%20Budgets-0B5FFF)
![Prometheus](https://img.shields.io/badge/Telemetry-Prometheus-E6522C?logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Experience-Grafana-F46800?logo=grafana&logoColor=white)
![Go](https://img.shields.io/badge/Implementation-Go-00ADD8?logo=go&logoColor=white)
![Continuous Validation](https://github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/actions/workflows/ci.yml/badge.svg)
![License](https://img.shields.io/badge/License-Apache%202.0-blue)

[Two-Minute Demo](#two-minute-demo) ·
[Delivery Flow](#delivery-flow) ·
[Reliability Model](#reliability-model) ·
[Core Capabilities](#core-capabilities) ·
[Adoption](#adoption) ·
[Operations](#operations-and-security) ·
[Documentation](#documentation) ·
[Contributing](#contributing)

## Project at a Glance

| Area | Delivered capability |
| --- | --- |
| Reliability contracts | Version-controlled SLIs, SLO targets, windows, ownership, runbooks, and decision policy |
| Validation | JSON Schema plus semantic and PromQL checks with precise field-level errors |
| Objective evaluation | Availability and latency attainment from provider-neutral good/total-event evidence |
| Error budgets | Consumption, remaining budget, and burn rate for every objective |
| Operational response | Generated multi-window alerts, Grafana dashboard, and linked runbooks |
| Delivery guidance | Explainable `proceed`, `caution`, `pause`, or `unknown` recommendations |
| Developer experience | One-command healthy-to-breach journey with automatic cleanup |
| Continuous validation | Race tests, static analysis, contract gates, and the complete Docker journey in CI |

## Two-Minute Demo

Prerequisites: Docker Desktop, Go 1.26 or newer, and `curl`.

```bash
./scripts/two-minute-demo.sh
```

The command validates the contract, starts the checkout demo, Prometheus, and
Grafana, proves healthy evidence produces `proceed`, generates a controlled
breach, verifies availability and latency burn-rate alerts, and proves the
resulting evidence produces `pause`.

The alert stage may wait for the contract's one-minute evaluation interval.
The script prints the SLO calculations and decision reasons, then always
removes its containers and network. The same journey runs in GitHub Actions.

## Delivery Flow

![Vaipex Service Reliability Control Plane flow](docs/images/vaipex-service-reliability-control-plane-flow.svg)

1. A service owner declares SLIs, SLOs, evaluation windows, ownership, and
   runbooks in a version-controlled reliability contract.
2. Automated controls validate its structure, semantics, and PromQL.
3. Prometheus-compatible telemetry supplies good- and total-event evidence.
4. The control plane calculates attainment, burn rate, and remaining budget.
5. Generated alerts, runbooks, and dashboards provide operational feedback.
6. The decision interface recommends `proceed`, `caution`, `pause`, or
   `unknown` and includes the evidence behind that result.
7. Delivery systems and accountable teams choose how to apply the guidance.

## Why This Project Exists

Teams often collect metrics without agreeing on what reliable service means.
The result is noisy alerting, subjective release decisions, and a gap between
product expectations and operational behavior.

This project demonstrates a platform-owned reliability model: service teams
declare intent through a supported contract, the platform evaluates every
service consistently, and transparent error-budget evidence guides delivery
and improvement. The control plane informs decisions; it does not hide the
evidence or replace human ownership.

## Reliability Model

| Term | Plain-language meaning | Example |
| --- | --- | --- |
| SLI | What the system measures | Proportion of successful requests |
| SLO | The internal reliability target | 99.9% successful requests over 30 days |
| Error budget | The unreliability permitted by the SLO | 0.1% of requests may fail in the window |
| Burn rate | How quickly that budget is being consumed | A burn rate of 2 consumes budget twice as fast as planned |
| SLA | An external commitment and its consequences | A contractual 99.9% availability commitment |

SLIs and SLOs are operational engineering tools. An SLA is a business or
customer commitment that should normally be informed by—not substituted for—the
internal reliability model.

## Core Capabilities

### Reliability contract and validation

The strict, versioned `ServiceReliability` contract connects service ownership
to availability and latency objectives, Prometheus evidence, burn-rate alert
policy, runbooks, and delivery thresholds.

```bash
go run ./cmd/reliabilityctl validate \
  contracts/examples/checkout-service.yaml
```

Expected result:

```text
valid: contracts/examples/checkout-service.yaml (service=checkout-service objectives=2)
```

The validator accepts YAML or JSON and combines the portable
[JSON Schema](contracts/service-reliability.schema.json) with semantic checks
for identifiers, duration ordering, adapter compatibility, PromQL, and policy
thresholds. Run the complete positive and negative contract gate with:

```bash
./scripts/validate-contracts.sh
```

### SLO evaluation and error budgets

Evaluate the included provider-neutral evidence:

```bash
go run ./cmd/reliabilityctl evaluate \
  --contract contracts/examples/checkout-service.yaml \
  --evidence contracts/examples/checkout-service-evidence.yaml
```

The result exposes target and attainment percentages, good and total events,
allowed and observed bad-event fractions, burn rate, budget consumption,
remaining budget, and `met`, `breached`, or `unknown` status. Missing, stale,
future-dated, misaligned, zero-traffic, and insufficient-sample evidence never
becomes false success.

### Live Prometheus evidence

```bash
./scripts/start-local.sh
./scripts/generate-traffic.sh
./scripts/evaluate-live.sh
```

The local platform exposes the demo service on `http://localhost:18080` and
Prometheus on `http://localhost:19090`. `reliabilityctl collect` renders the
contract's window-aware PromQL, queries Prometheus, and writes the same portable
evidence consumed by the evaluation engine.

### Dashboard, alerts, and runbooks

Open the provisioned Grafana dashboard after starting the stack:

```text
http://localhost:13000/d/vaipex-service-reliability
```

The contract generates recording rules and paired-window burn-rate alerts
rather than relying on separately maintained definitions. The dashboard shows
SLO attainment, budget remaining, burn rate, active alerts, request outcomes,
and evaluation windows.

```bash
./scripts/generate-breach.sh
./scripts/verify-alerts.sh
```

Fast-burn alerts pair 5-minute and 1-hour windows. Slow-burn alerts pair
30-minute and 6-hour windows. Generated alerts carry service, objective, team,
severity, and response-runbook context.

### Explainable delivery decisions

```bash
go run ./cmd/reliabilityctl decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence contracts/examples/checkout-service-evidence.yaml
```

The `ReliabilityDecision` JSON includes the recommendation, active severities,
policy thresholds, reason codes, objective IDs, and triggering values. Valid
recommendations exit successfully because the interface is advisory; consumers
choose how to enforce `caution`, `pause`, and `unknown`.

Explore every deterministic state:

```bash
# caution: objectives are met, but little budget remains
go run ./cmd/reliabilityctl decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence testdata/evidence/degraded.yaml

# pause: error budgets are exhausted
go run ./cmd/reliabilityctl decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence testdata/evidence/exhausted.yaml

# unknown: evidence is missing or stale
go run ./cmd/reliabilityctl decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence testdata/evidence/missing-and-stale.yaml
```

## Adoption

Start with one owned service and one user-centered availability or latency
journey. Adapt the contract and queries, connect production telemetry, route
generated alerts through existing incident processes, and publish decisions in
advisory mode before considering enforcement.

Reuse the contract, evidence model, calculations, generated rules, decision
output, and automated verification. Replace the sample identities, metric
names, thresholds, containers, and anonymous local access with
organization-managed services and controls. Follow the
[Adoption Guide](docs/adoption.md) for the complete staged approach.

## Operations and Security

```bash
./scripts/start-local.sh
./scripts/stop-local.sh
```

All published demo ports bind to `127.0.0.1`. The sample service runs as a
non-root user with a read-only filesystem, dropped Linux capabilities, and
`no-new-privileges`. Images and GitHub Actions are pinned immutably, CI uses
read-only repository permissions, and dependency updates are automated.

This is a local reference environment, not a production distribution. It does
not provide production identity, TLS, authorization, tenant isolation,
durability, high availability, backup, or external secret management. See the
[Operations Guide](docs/operations.md) and [Security Policy](SECURITY.md)
before adapting it.

## Documentation

| Guide | Purpose |
| --- | --- |
| [Reliability contract](docs/reliability-contract.md) | Contract fields, semantics, policy, and compatibility |
| [Evaluation engine](docs/evaluation-engine.md) | SLO formulas, evidence rules, and fixtures |
| [Live Prometheus](docs/live-prometheus.md) | Telemetry adapter and live evidence flow |
| [Dashboards and alerts](docs/dashboards-and-alerts.md) | Generated rules, alert strategy, dashboard, and runbooks |
| [Decision interface](docs/decision-interface.md) | Recommendation precedence, reasons, and delivery integration |
| [Continuous validation](docs/continuous-validation.md) | CI gates and end-to-end journey |
| [Adoption guide](docs/adoption.md) | Incremental organizational rollout |
| [Operations guide](docs/operations.md) | Local lifecycle, diagnostics, cleanup, and production boundaries |

## Design Principles

- Reliability definitions are explicit, versioned, reviewable, and owned.
- Every status and recommendation is traceable to objective evidence.
- Missing or stale telemetry is `unknown`, never healthy.
- Alerts focus on meaningful budget consumption rather than isolated noise.
- Delivery controls are transparent, reversible, and policy-driven.
- Provider-neutral interfaces are preserved where practical.
- Reliability signals describe services and systems, never individuals.

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md), follow the
[Community Code of Conduct](CODE_OF_CONDUCT.md), and report vulnerabilities
privately according to [SECURITY.md](SECURITY.md).

Licensed under the [Apache License 2.0](LICENSE).
