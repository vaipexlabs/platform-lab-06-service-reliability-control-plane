# Sustained Error-Budget Burn Runbook

Use this runbook when a ticket-severity reliability alert reports sustained
budget consumption across the configured short and long windows.

## Interpretation

A slow burn may not require immediate incident response, but it indicates that
the service will miss its objective if the current reliability level
continues. It should become planned reliability work with a named owner and
review date.

## Investigation

1. Confirm the signal across both windows and check evidence freshness.
2. Segment failures or latency by route, dependency, region, and release.
3. Compare the trend with deployment, demand, and dependency changes.
4. Estimate time to budget exhaustion at the current burn rate.
5. Identify the smallest systemic improvement likely to reduce the burn.

## Action

- Create a reliability work item owned by the service team.
- Link the dashboard, objective, observed burn rate, and supporting evidence.
- Consider reducing delivery risk while remaining budget is low.
- Escalate to the fast-burn process if user impact or burn rate increases.

Close the work only after the long-window signal demonstrates sustained
recovery.

