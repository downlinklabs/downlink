# API Contract: Cluster API Gateway

**Base path**: `/api/v1`
**Auth**: All endpoints require `Authorization: Bearer <api-key>` header. Requests without a valid key receive `401 Unauthorized`.
**Content-Type**: All responses are `application/json`.

---

## GET /api/v1/tests

List all SyntheticTests across all namespaces.

**Response 200**
```json
{
  "tests": [
    {
      "cluster": "default",
      "namespace": "downlink",
      "name": "checkout-flow",
      "schedule": "*/5 * * * *",
      "lastRunPhase": "Succeeded",
      "lastRunAt": "2026-11-28T15:00:00Z"
    }
  ]
}
```

**Response 401** — missing or invalid API key
```json
{ "error": "unauthorized", "code": "unauthorized" }
```

---

## GET /api/v1/tests/{namespace}/{name}

Get a single SyntheticTest.

**Path parameters**: `namespace`, `name`

**Response 200** — same shape as a single item from the list response

**Response 404**
```json
{ "error": "SyntheticTest downlink/checkout-flow not found", "code": "not_found" }
```

---

## GET /api/v1/tests/{namespace}/{name}/runs

List SyntheticTestRuns for a specific SyntheticTest, ordered by `startedAt` descending.

**Path parameters**: `namespace`, `name`

**Query parameters**:
- `limit` (optional, integer, default 20, max 100) — number of runs to return

**Response 200**
```json
{
  "runs": [
    {
      "cluster": "default",
      "namespace": "downlink",
      "name": "checkout-flow-abc12",
      "testName": "checkout-flow",
      "phase": "Succeeded",
      "startedAt": "2026-11-28T15:00:00Z",
      "finishedAt": "2026-11-28T15:00:45Z",
      "artifactUrl": "s3://my-bucket/checkout-flow-abc12/playwright-report",
      "message": null
    }
  ]
}
```

**Response 404** — parent SyntheticTest not found
```json
{ "error": "SyntheticTest downlink/checkout-flow not found", "code": "not_found" }
```

---

## GET /api/v1/runs/{namespace}/{name}

Get a single SyntheticTestRun by name.

**Path parameters**: `namespace`, `name`

**Response 200** — single run object (same shape as list item)

**Response 404**
```json
{ "error": "SyntheticTestRun downlink/checkout-flow-abc12 not found", "code": "not_found" }
```

---

## POST /api/v1/tests/{namespace}/{name}/trigger

Trigger an on-demand run of a SyntheticTest. Submits a one-shot Argo Workflow and returns the resulting SyntheticTestRun name.

**Path parameters**: `namespace`, `name`

**Request body**: empty

**Response 202 Accepted**
```json
{
  "cluster": "default",
  "namespace": "downlink",
  "runName": "checkout-flow-xyz99"
}
```

**Response 404** — SyntheticTest not found
```json
{ "error": "SyntheticTest downlink/checkout-flow not found", "code": "not_found" }
```

**Response 409 Conflict** — a run is already in progress for this test
```json
{ "error": "a run for checkout-flow is already in progress", "code": "conflict" }
```

---

## Health

**GET /healthz** — liveness probe (no auth required). Returns `200 OK` with body `ok`.

**GET /readyz** — readiness probe (no auth required). Returns `200 OK` with body `ok` when the Store is reachable.
