# 10. Pull-based sync agent for control plane connectivity

Date: 2026-11-24

## Status

Accepted (not yet implemented)

## Context

For a hosted control plane to display run history and artifacts from customer clusters, it needs access to
`SyntheticTestRun` objects. Three options were considered:

1. **Operator phones home** — the operator pushes run events to the control plane API directly. Simple but
   couples the OSS operator to the SaaS, undermining the open-core story.
2. **Pull-based sync agent** — a separate, optional component runs in the customer's cluster, reads
   `SyntheticTestRun` objects, and ships them to the control plane. The operator itself has zero SaaS coupling.
3. **Control plane calls in** — the hosted SaaS uses a customer-provided kubeconfig or token to read CRs
   directly. Poor security posture; impractical for firewalled clusters.

## Decision

The sync agent is a separate deployable component (not part of the operator). It is the only component that
knows about the control plane. The operator remains a pure Kubernetes controller with no outbound SaaS calls,
preserving its viability as a standalone OSS tool.

## Consequences

- The operator can be used indefinitely without ever connecting to a control plane.
- The sync agent is the clear product boundary between free and paid.
- Customers in air-gapped or firewalled environments can still run the operator; connectivity to the control
  plane is opt-in and agent-mediated.
- The sync agent must handle back-pressure, retries, and deduplication independently of the operator.
