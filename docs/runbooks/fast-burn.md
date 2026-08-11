# Fast Error-Budget Burn Runbook

Use this runbook when a page-severity reliability alert reports that both the
short and long burn-rate windows exceed the contract threshold.

## User Impact

A fast burn means the service is consuming its error budget many times faster
than the SLO permits. Without intervention, the objective may be exhausted
quickly. Treat the signal as user-impacting until evidence shows otherwise.

## First 10 Minutes

1. Acknowledge the alert and identify its `service` and `objective` labels.
2. Confirm both burn-rate windows in the Grafana reliability dashboard.
3. Check request volume to rule out a no-traffic or low-sample artifact.
4. Inspect recent deployments, feature flags, dependencies, and infrastructure
   changes for temporal correlation.
5. Compare failure or latency dimensions to identify the affected route,
   region, dependency, or customer cohort.
6. Stop or roll back the suspected change when evidence supports that action.
7. Record the mitigation and current user impact in the incident channel.

## Recovery

- Confirm the source service metric is current and Prometheus is scraping it.
- Verify the short-window burn rate falls first, followed by the long window.
- Keep the incident open until the service is stable and the alert clears.
- Do not reset counters or modify SLO targets to make the alert disappear.

## Follow-Up

Capture the contributing system conditions, detection quality, mitigation,
and any platform improvement. Review whether the SLO and alert policy remain
appropriate using evidence—not the desire to avoid an alert.

