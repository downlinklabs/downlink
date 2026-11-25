# 9. HTTP trigger endpoint for on-demand and CI-triggered runs

Date: 2026-11-24

## Status

Accepted

## Context

`BuildWorkflow` exists on the `WorkflowBuilder` interface, but there was no mechanism to call it externally. The
annotation-based escape hatch and kubectl plugin approaches were considered, but neither works well for CI
pipelines that need to trigger a run after a deployment and block until pass/fail — particularly from systems
that don't have kubectl or cluster credentials available.

## Decision

The operator exposes an authenticated HTTP endpoint (`POST /trigger/{namespace}/{name}`) that submits a
one-shot Argo `Workflow` from a named `SyntheticTest` and returns a run ID. Auth uses a static API key
generated at install time and stored as a Kubernetes Secret. The endpoint is exposed via an optional ingress
in the Helm chart.

## Consequences

- CI pipelines can trigger runs and poll for `SyntheticTestRun` completion without kubectl.
- The operator gains a public HTTP surface that requires TLS and key rotation discipline.
- This endpoint is the natural integration point for the future hosted control plane — the control plane
  calls the same endpoint to trigger runs in customer clusters.
- OIDC/JWT auth is left as a future enterprise hardening option; API key is sufficient for v1.
