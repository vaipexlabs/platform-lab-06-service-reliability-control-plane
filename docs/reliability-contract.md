# Service Reliability Contract

The `ServiceReliability` contract gives service teams and platform systems one
version-controlled definition of reliability intent. It describes ownership,
objectives, telemetry evidence, error-budget alerts, and the policy used to
form delivery recommendations.

The current API is `reliability.vaipex.io/v1alpha1`. The alpha designation
makes evolution explicit: incompatible changes require a new API version and
must not silently reinterpret an existing contract.

## Contract Boundaries

The contract owns:

- stable service identity and operational ownership;
- SLI category and the queries that produce good and total event counts;
- SLO target, evaluation window, and minimum evidence requirement;
- multi-window burn-rate alert intent; and
- thresholds used to recommend proceed, caution, or pause.

The contract does not own:

- telemetry collection or retention;
- deployment credentials or permission to change a cluster;
- contractual SLA penalties;
- incident escalation details beyond a linked runbook; or
- automatic remediation of an unhealthy service.

Those boundaries keep the control plane focused on reliable, explainable
decisions rather than turning it into a monitoring or deployment platform.

## Document Shape

| Field | Responsibility |
| --- | --- |
| `apiVersion` and `kind` | Select the contract semantics understood by the control plane |
| `metadata` | Provide a stable name and optional discovery labels |
| `spec.service` | Identify the display name, owning team, criticality tier, source, and runbook |
| `spec.evaluation` | Define evaluation cadence and the maximum acceptable telemetry age |
| `spec.objectives` | Declare one or more indicators, targets, windows, and burn-rate policies |
| `spec.decisionPolicy` | Translate remaining budget and active alerts into delivery guidance |

See the [checkout-service example](../contracts/examples/checkout-service.yaml)
for a complete contract.

## Indicator Interface

Every indicator produces two comparable values:

- `goodEventsQuery` returns events that met the user-visible expectation.
- `totalEventsQuery` returns all valid events in the same population.

The evaluator calculates:

```text
SLI attainment (%) = good events / total events × 100
```

The contract is adapter-oriented rather than locked to one monitoring vendor.
`provider` selects the telemetry adapter and `queryLanguage` describes its
query syntax. The reference journey starts with `prometheus` and `promql`, but
the contract shape allows another adapter to implement the same good-events /
total-events interface.

Both queries must use the same service, environment, and event population;
must resolve to one numeric value; and must use compatible units. The platform
validator enforces supported provider/language pairs. PromQL templates must
include `$window`; the collector replaces it with the objective window before
parsing and execution. This ensures the returned counts describe the same
window used by the error-budget calculation. The evaluator treats zero
traffic, insufficient samples, stale telemetry, and query failure as `unknown`,
never as a successful objective.

## Objective and Error-Budget Semantics

`target` is the required percentage of good events over `window`. For a 99.9%
target, the allowed bad-event fraction is 0.1%. Targets must be below 100%
because a 100% target has no finite error budget and cannot produce a useful
burn rate.

```text
allowed bad fraction = 1 - target fraction
observed bad fraction = 1 - observed good fraction
burn rate = observed bad fraction / allowed bad fraction
remaining budget (%) = (1 - burn rate) × 100
```

A burn rate of `1` consumes the budget at exactly the sustainable rate. A burn
rate above `1` consumes it too quickly. Remaining budget may become negative;
that is useful evidence that the objective has been exceeded rather than a
value to hide by clamping it to zero.

Each alert combines a short and a long observation window. Requiring both
windows to exceed the threshold reduces sensitivity to isolated spikes while
still detecting meaningful fast or sustained consumption.

## Delivery Decision Semantics

The initial decision model will be deliberately small and explainable:

| Recommendation | Meaning |
| --- | --- |
| `proceed` | Objectives have usable evidence and no caution or pause condition is active |
| `caution` | Remaining budget is below the configured caution threshold |
| `pause` | Budget is exhausted or a configured burn-rate severity is active |
| `unknown` | Evidence is missing, stale, insufficient, or cannot be evaluated |

These are recommendations for a delivery policy to consume. The control plane
does not deploy code, and adopting organizations decide whether a
recommendation is advisory or enforced.

## Schema and Semantic Validation

The [JSON Schema](../contracts/service-reliability.schema.json) defines the
structural contract: required fields, supported values, types, duration
syntax, and strict rejection of unknown properties.

Some rules require semantic validation and belong to the next milestone:

- objective and alert names must be unique within their parent scope;
- long burn-rate windows must be longer than their short windows;
- evaluation cadence must be shorter than objective and alert windows;
- pause thresholds must not be less conservative than caution thresholds;
- the provider and query-language combination must be supported; and
- PromQL queries must be syntactically valid and resolve to safe instant-query
  expressions.

Separating structural and semantic validation keeps the schema portable while
still producing precise, human-readable feedback through the platform tooling.
