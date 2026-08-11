# Continuous Validation

The repository validates both the control-plane code and its complete user
journey on every pull request and push to `main`. The workflow can also be run
manually through GitHub Actions.

## Quality Job

The fast quality job checks:

- Go formatting and static analysis;
- race-enabled unit and CLI tests;
- supported and intentionally invalid reliability contracts;
- shell syntax;
- Grafana dashboard JSON; and
- the Docker Compose model.

The job fails before the end-to-end environment starts if any foundational
contract or implementation check is broken.

## End-to-End Job

The end-to-end job runs the same public command documented for local users:

```bash
./scripts/two-minute-demo.sh
```

The demo performs five observable stages:

1. Validate the example reliability contract.
2. Start the checkout demo, Prometheus, and Grafana.
3. Generate healthy traffic and assert a `proceed` recommendation.
4. Generate a controlled breach and verify the availability and latency
   fast-burn alerts.
5. Evaluate breached evidence and assert a `pause` recommendation.

The script always removes the project containers and network on exit, whether
the journey passes or fails. It also starts from a clean project stack so old
Prometheus samples cannot affect the deterministic result.

## Security and Reproducibility

The workflow has read-only repository permissions, uses immutable commit SHAs
for third-party actions, applies job timeouts, and requires no repository
secrets. Runtime container images are pinned by digest in the Compose model.

## Local Validation

Prerequisites are Docker Desktop, Go 1.26 or newer, and `curl`:

```bash
./scripts/two-minute-demo.sh
```

The alert stage may wait for the contract's one-minute evaluation interval.
The complete journey is designed to finish within two minutes on a warm local
Docker environment and within the five-minute CI timeout on a fresh runner.
