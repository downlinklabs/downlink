# 2. Argo Workflows as the execution engine

Date: 2026-10-21

## Status

Accepted

## Context

Synthetic tests need to run on a schedule inside the cluster. The obvious Kubernetes primitive is a `CronJob`,
but raw CronJobs have no retry policy, no workflow-level artifacts, no execution history, and no UI. We needed something
that could grow with the platform — DAG support, per-step artifact collection, and a developer-facing UI without building one.

## Decision

We use Argo Workflows (`CronWorkflow`) as the scheduler and execution engine. The operator reconciles each `SyntheticTest` CR
into an Argo `CronWorkflow`. Argo handles scheduling, pod lifecycle, retries, and artifact upload.

## Consequences

- Argo Workflows is a cluster dependency. Users must either accept the bundled chart or bring their own instance.
- We get artifact storage, retry policy, and a workflow UI for free.
- The operator speaks Argo's type system (`wfv1.*`), coupling it to the Argo API version.
- On-demand and event-driven runs are a straightforward extension: build a one-shot `Workflow` from the same spec.
