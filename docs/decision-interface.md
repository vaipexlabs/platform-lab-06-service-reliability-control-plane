# Explainable Reliability Decision Interface

The decision interface converts evaluated service reliability into concise,
machine-readable delivery guidance. It is deliberately advisory: the control
plane explains the recommendation, while an adopting organization decides
whether a pipeline displays, reviews, or enforces it.

## Inputs

`reliabilityctl decide` accepts:

- a validated `ServiceReliability` contract containing the decision policy;
- a provider-neutral evidence document used by the existing SLO evaluator;
  and
- zero or more active burn-rate alert severities supplied with repeated
  `--active-severity` flags.

The command validates the contract and evidence, runs the existing evaluation
engine, and then applies the declared policy. It does not recalculate SLOs
through a second implementation.

## Recommendations

| Recommendation | Condition |
| --- | --- |
| `pause` | Any objective has remaining budget at or below the pause threshold, or an active severity matches the pause policy |
| `unknown` | No pause condition exists, but at least one objective lacks usable evidence |
| `caution` | No pause or unknown condition exists, but remaining budget is below the caution threshold |
| `proceed` | Every objective has usable evidence and no policy threshold is active |

Precedence is `pause` → `unknown` → `caution` → `proceed`. A known severe
condition therefore cannot be softened by missing evidence elsewhere.

## Example

```bash
go run ./cmd/reliabilityctl decide \
  --contract contracts/examples/checkout-service.yaml \
  --evidence testdata/evidence/exhausted.yaml \
  --active-severity page
```

The output is a `ReliabilityDecision` JSON document containing:

- the service and evaluation timestamp;
- the final recommendation;
- normalized active severities;
- the exact caution and pause policy used; and
- structured reasons containing codes, objective IDs, observed values, and
  thresholds where applicable.

Representative reason codes are:

| Code | Meaning |
| --- | --- |
| `within_policy` | All evaluated objectives remain within the declared policy |
| `low_error_budget` | Remaining budget crossed the caution threshold |
| `error_budget_exhausted` | Remaining budget crossed the pause threshold |
| `active_burn_rate_severity` | An active alert severity matched the pause policy |
| `unknown_evidence` | An objective could not be evaluated safely |

## Delivery-System Integration

All valid recommendations exit with status `0` and are distinguished by the
JSON `recommendation` field. Validation, malformed evidence, or operational
errors return a non-zero exit code. Keeping policy output separate from process
exit status lets consumers choose their own enforcement model:

- display the recommendation during pull-request review;
- require approval when the recommendation is `caution`;
- stop promotion when the recommendation is `pause`; or
- fail closed or request human review when the recommendation is `unknown`.

The interface never deploys, rolls back, or changes a service. Its purpose is
to make reliability evidence portable and the resulting policy decision easy
to inspect and automate.
