# 5. SyntheticTest CRD and operator pattern

Date: 2026-10-21

## Status

Accepted

## Context

Tests need a management surface — a way to define, update, and delete them without editing Helm values or
touching Argo directly. Options considered: Helm values (inflexible, requires chart upgrade to change a schedule),
a custom REST API (another service to build and secure), or Kubernetes Custom Resources (native to the cluster,
GitOps-friendly, `kubectl`-manageable).

## Decision

We model synthetic tests as Kubernetes Custom Resources (`SyntheticTest`) and run a kubebuilder operator to
reconcile them. The operator watches `SyntheticTest` objects and owns the `CronWorkflow`, `ConfigMap`, `ServiceAccount`,
`Role`, and `RoleBinding` that implement each test. A second CRD (`SyntheticTestRun`) mirrors completed workflow runs as
structured objects so the UI has a typed API without calling Argo directly.

## Consequences

- Tests are version-controlled as YAML, fit naturally into GitOps workflows, and are managed with `kubectl apply`.
- The operator adds operational complexity: it must be deployed, monitored, and upgraded alongside the chart.
- `SyntheticTestRun` decouples the UI from Argo's API, but requires the operator to keep run records in sync with workflow state.
- The kubebuilder scaffold (controller-runtime) is the standard Go operator pattern; it has good testing primitives and a mature ecosystem.
