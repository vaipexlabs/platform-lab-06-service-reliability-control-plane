# Adoption Guide

Adopt the model incrementally. The goal is a shared reliability language and
repeatable evidence—not copying a local Docker stack into production.

## 1. Select a Service

Begin with one service that has clear ownership, meaningful user traffic, and
request telemetry. Choose a user journey the team can explain, such as
successful checkouts or requests completed within a latency threshold.

## 2. Define the Reliability Contract

Copy the example contract and replace:

- service identity, repository, tier, team, contact, and runbook;
- availability and latency definitions;
- PromQL selectors and thresholds;
- objective targets and rolling windows;
- minimum sample requirements;
- burn-rate windows and severities; and
- caution and pause policy thresholds.

Validate every change:

```bash
go run ./cmd/reliabilityctl validate path/to/service-reliability.yaml
```

Targets should reflect user expectations and product tradeoffs. Do not choose
an SLO merely because current telemetry happens to meet it.

## 3. Connect Production Telemetry

Ensure good- and total-event queries use the same population and cannot count
good events more than once. Keep labels stable, avoid unbounded cardinality,
and test behavior for no traffic, partial failure, missing data, and delayed
scrapes. The reference adapter uses Prometheus, but the evidence interface is
provider-neutral.

## 4. Introduce Alerts and Runbooks

Generate recording and burn-rate rules from the validated contract. Route
page and ticket severities through the organization's existing incident
process. Replace the example runbooks with service-specific diagnostics,
rollback guidance, ownership, and escalation paths.

## 5. Roll Out Decisions Gradually

Start by publishing `ReliabilityDecision` results without blocking delivery.
Review false positives, unknown evidence, budget behavior, and operator
feedback. Then consider progressive policy:

1. display all recommendations;
2. require human review for `caution`;
3. require review or pause promotion for `unknown`; and
4. enforce `pause` only after teams trust the data and recovery path.

The control plane recommends; organizational policy determines enforcement.

## 6. Establish Governance

Review service objectives on a predictable cadence and after material product
or architecture changes. Track whether alerts led to useful action, whether
runbooks accelerated recovery, and whether delivery decisions were supported
by trustworthy evidence. Never use service reliability signals to rank or
evaluate individual contributors.

## What to Reuse

- the versioned contract and schema-validation pattern;
- provider-neutral good-events and total-events evidence;
- error-budget and burn-rate calculations;
- generated multi-window alert rules;
- explainable decision output; and
- automated healthy and breach verification.

Replace the local containers, anonymous access, example ownership, metric
names, and example thresholds with organization-managed equivalents.
