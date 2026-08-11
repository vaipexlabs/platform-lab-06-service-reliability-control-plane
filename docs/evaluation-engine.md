# SLO Evaluation and Error Budgets

The evaluation engine separates telemetry collection from reliability math.
An adapter converts provider-specific query results into a versioned
`ReliabilityEvidence` snapshot containing good and total event counts. The
engine then combines that evidence with a validated `ServiceReliability`
contract.

This boundary makes calculations deterministic, testable, and explainable. A
future Prometheus adapter can change without changing SLO or error-budget
semantics.

## Calculation Model

For each objective:

```text
attainment = good events / total events
allowed bad fraction = 1 - target fraction
observed bad fraction = 1 - attainment
burn rate = observed bad fraction / allowed bad fraction
budget consumed (%) = burn rate × 100
budget remaining (%) = (1 - burn rate) × 100
```

A negative remaining budget is retained as evidence that the objective was
exceeded. It is not clamped to zero.

## Objective States

| State | Meaning |
| --- | --- |
| `met` | Attainment is at or above the target and the evidence is usable |
| `breached` | Attainment is below the target and the evidence is usable |
| `unknown` | The engine cannot safely calculate the objective |

The engine returns `unknown` when evidence is missing, stale, from the future,
misaligned with the objective window, contains no traffic, or does not meet
the contract's minimum sample count. Unknown is deliberately distinct from
healthy.

## Evidence Rules

- The evidence service name must match the reliability contract.
- Objective evidence identifiers must be unique and declared by the contract.
- Good events must not exceed total events.
- The evidence window must match the objective window and end at the declared
  evaluation time.
- The observation timestamp must not be in the future or older than the
  contract's `dataFreshness` limit.
- Every output objective follows contract order, so a missing evidence item is
  visible rather than silently omitted.

The [evidence JSON Schema](../contracts/reliability-evidence.schema.json)
provides strict structural validation. The engine applies the relationship and
time rules after schema validation.

## Deterministic Demonstration

```bash
go run ./cmd/reliabilityctl evaluate \
  --contract contracts/examples/checkout-service.yaml \
  --evidence contracts/examples/checkout-service-evidence.yaml
```

The representative snapshot produces two met objectives. Both have a burn
rate of `0.5`, which means half of each objective's 30-day error budget has
been consumed.

