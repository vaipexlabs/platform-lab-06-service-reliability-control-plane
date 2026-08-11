# Security Policy

## Reporting a Vulnerability

Do not open a public issue for a suspected vulnerability. Email
`vaipex.labs@gmail.com` with:

- the affected component and version or commit;
- reproduction steps or a minimal proof of concept;
- the potential impact; and
- any suggested mitigation.

Please avoid accessing data that does not belong to you, disrupting shared
systems, or publishing details before a fix can be evaluated. Vaipex Labs will
acknowledge the report, assess its scope, and coordinate disclosure when the
finding is confirmed.

## Supported Version

Security fixes target the latest commit on `main`. This project does not
currently maintain supported release branches.

## Reference-Implementation Boundary

This repository is an educational and architectural reference, not a managed
service or production-ready distribution. The local stack:

- binds its published ports to `127.0.0.1`;
- enables anonymous Grafana viewing for local demonstration only;
- uses ephemeral local Prometheus and Grafana data;
- does not include TLS, authentication, authorization, tenant isolation,
  backups, high availability, or external secret management; and
- must not be exposed to untrusted networks.

Organizations adopting the patterns are responsible for their own threat
model, identity controls, network policy, encryption, data retention, supply
chain controls, and operational support model.
