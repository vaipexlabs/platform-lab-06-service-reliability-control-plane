# Contributing

Contributions that improve the clarity, portability, safety, or operational
value of the reference implementation are welcome.

## Before You Start

- Use a public issue to discuss substantial behavior or contract changes.
- Use the private process in [SECURITY.md](SECURITY.md) for vulnerabilities.
- Keep decisions explainable and provider-neutral where practical.
- Preserve the boundary between service reliability signals and individual
  performance measurement.

## Development Workflow

1. Fork the repository and create a focused branch.
2. Make the smallest coherent change.
3. Add or update tests and documentation with the implementation.
4. Run the quality checks:

   ```bash
   gofmt -w cmd internal
   go vet ./...
   go test -race ./...
   ./scripts/validate-contracts.sh
   bash -n scripts/*.sh
   docker compose -f deploy/local/compose.yaml config --quiet
   ```

5. Run `./scripts/two-minute-demo.sh` when changing contracts, telemetry,
   rules, dashboards, alerts, decisions, containers, or demo scripts.
6. Open a pull request that explains the problem, approach, user impact, and
   validation performed.

## Contract Compatibility

The `ServiceReliability` schema is a public interface. Changes must either
remain compatible with `v1alpha1` or introduce a clearly documented new API
version. Examples, semantic validation, generated rules, evaluation, decision
output, and adoption guidance must remain aligned.

## Contribution Terms

Unless stated otherwise, contributions are submitted under the repository's
[Apache License 2.0](LICENSE), as described by section 5 of that license.
