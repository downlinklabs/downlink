# Quickstart Validation Guide: Cluster API Gateway

This guide proves the feature works end-to-end. Run these scenarios after deploying the updated operator.

## Prerequisites

- Downlink operator deployed with the `--api-key` and `--cluster-id` flags set
- At least one `SyntheticTest` applied to the cluster
- `curl` and `kubectl` available
- API key stored in `API_KEY` environment variable
- Operator HTTP port forwarded: `kubectl port-forward -n downlink svc/downlink-api 8082:8082`

---

## Scenario 1: List SyntheticTests

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  http://localhost:8082/api/v1/tests | jq .
```

**Expected**: JSON object with a `tests` array. Each entry has `cluster`, `namespace`, `name`, `schedule`, `lastRunPhase`, `lastRunAt`.

**Failure indicators**: 401 (wrong key), empty array when tests exist (cache not synced — wait 5s and retry).

---

## Scenario 2: Get a specific SyntheticTest

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  http://localhost:8082/api/v1/tests/downlink/checkout-flow | jq .
```

**Expected**: Single test object. `lastRunPhase` is `null` if the test has never run.

**Failure indicator**: 404 means the test name or namespace is wrong.

---

## Scenario 3: Trigger an on-demand run and poll for completion

```bash
# Trigger
RUN=$(curl -s -X POST -H "Authorization: Bearer $API_KEY" \
  http://localhost:8082/api/v1/tests/downlink/checkout-flow/trigger | jq -r .runName)
echo "Run: $RUN"

# Poll until complete
while true; do
  PHASE=$(curl -s -H "Authorization: Bearer $API_KEY" \
    http://localhost:8082/api/v1/runs/downlink/$RUN | jq -r .phase)
  echo "Phase: $PHASE"
  [[ "$PHASE" == "Succeeded" || "$PHASE" == "Failed" ]] && break
  sleep 5
done
```

**Expected**: Run transitions from `Running` to `Succeeded` or `Failed` within the test's normal execution time.

---

## Scenario 4: List run history

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  "http://localhost:8082/api/v1/tests/downlink/checkout-flow/runs?limit=5" | jq .
```

**Expected**: Array of up to 5 runs ordered by `startedAt` descending. Runs with artifact storage configured include a non-null `artifactUrl`.

---

## Scenario 5: Auth rejection

```bash
curl -s http://localhost:8082/api/v1/tests
```

**Expected**: `401` response with `{"error":"unauthorized","code":"unauthorized"}`.

---

## Scenario 6: Health probes (no auth)

```bash
curl -s http://localhost:8082/healthz   # → 200 "ok"
curl -s http://localhost:8082/readyz    # → 200 "ok"
```

These must succeed without an API key — they are used by Kubernetes liveness/readiness probes.
