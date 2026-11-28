# Data Model: Cluster API Gateway

These are the read-model view types returned by the API. They are projections of the existing CRDs — not new storage entities.

---

## SyntheticTestView

A flattened read projection of a `SyntheticTest` CR, enriched with last-run summary from its most recent `SyntheticTestRun`.

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `cluster` | string | config (`--cluster-id`) | Stable identifier for the originating cluster |
| `namespace` | string | `SyntheticTest.metadata.namespace` | |
| `name` | string | `SyntheticTest.metadata.name` | |
| `schedule` | string | `SyntheticTest.spec.schedule` | POSIX cron expression |
| `lastRunPhase` | string \| null | `SyntheticTestStatus.lastRunPhase` | `Succeeded`, `Failed`, `Running`, or `null` if never run |
| `lastRunAt` | RFC3339 \| null | `SyntheticTestStatus.lastRunAt` | Timestamp of the most recent run completion |

**Relationships**: A `SyntheticTestView` has zero or more `SyntheticTestRunView` records (one per completed Argo Workflow execution).

---

## SyntheticTestRunView

A flattened read projection of a `SyntheticTestRun` CR.

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `cluster` | string | config (`--cluster-id`) | |
| `namespace` | string | `SyntheticTestRun.metadata.namespace` | |
| `name` | string | `SyntheticTestRun.metadata.name` | Matches the Argo Workflow name |
| `testName` | string | `SyntheticTestRun.spec.name` | The parent SyntheticTest name |
| `phase` | string | `SyntheticTestRun.status.phase` | `Succeeded`, `Failed`, `Running` |
| `startedAt` | RFC3339 \| null | `SyntheticTestRun.spec.startedAt` | |
| `finishedAt` | RFC3339 \| null | `SyntheticTestRun.status.finishedAt` | |
| `artifactUrl` | string \| null | `SyntheticTestRun.status.artifactURL` | S3/GCS URI; null if artifact storage not configured |
| `message` | string \| null | `SyntheticTestRun.status.message` | Argo failure message; null on success |

---

## TriggerResult

Returned by `POST /api/v1/tests/{namespace}/{name}/trigger`.

| Field | Type | Notes |
|-------|------|-------|
| `cluster` | string | |
| `namespace` | string | |
| `runName` | string | The name of the created `SyntheticTestRun` (matches the submitted Argo Workflow name) |

---

## ErrorResponse

Returned on all error paths.

| Field | Type | Notes |
|-------|------|-------|
| `error` | string | Human-readable error message |
| `code` | string | Machine-readable code: `not_found`, `unauthorized`, `internal`, `conflict` |

---

## State Transitions

`SyntheticTestRunView.phase` follows Argo Workflow phases:

```
(created) → Running → Succeeded
                   ↘ Failed
```

The API does not synthesise intermediate states — it reflects exactly what the `SyntheticTestRun` CR contains.
