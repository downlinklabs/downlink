# 6. WorkflowBuilder interface for CronWorkflow construction

Date: 2026-10-21

## Status

Accepted

## Context

The `SyntheticTestReconciler` originally constructed Argo `CronWorkflow` structs inline. This made the reconciler
hard to unit test (tests required the full Argo type graph), and it left no clean path for on-demand or event-driven
workflow execution, which would need the same construction logic without a schedule.

## Decision

We extract all Argo object construction into a `WorkflowBuilder` interface with two methods: `BuildCronWorkflow`
and `BuildWorkflow`. The reconciler holds a `WorkflowBuilder` via constructor injection and calls it in place of
all inline struct literals. The production `DefaultWorkflowBuilder` is stateless. A `FakeWorkflowBuilder` enables
isolated reconciler tests.

## Consequences

- The reconciler contains zero Argo struct literals; it only applies what the builder returns.
- Reconciler tests inject a fake builder and assert on call arguments, with no live cluster required.
- `BuildWorkflow` positions the platform for manual triggers and webhook-driven runs without duplicating construction logic.
- The interface must be kept in sync as the `SyntheticTest` spec evolves; new fields need to be handled in `DefaultWorkflowBuilder`.
