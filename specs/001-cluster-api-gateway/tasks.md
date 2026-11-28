---
description: "Task list for Cluster API Gateway feature implementation"
---

# Tasks: Cluster API Gateway

**Input**: `specs/001-cluster-api-gateway/`

**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Contract**: [contracts/api.md](./contracts/api.md) | **Data model**: [data-model.md](./data-model.md)

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this belongs to ([US1], [US2], [US3])

## Path Conventions

- Operator source: `operator/`
- New shared builder package: `operator/pkg/builder/`
- New API binary: `api/`
- Helm chart: `charts/downlink/`

---

## Phase 1: Setup

**Purpose**: Establish the Go workspace, create the `api/` module skeleton, and move `WorkflowBuilder` out of `internal/` so both modules can import it.

- [ ] T001 Create `go.work` at repo root with `go work init ./operator ./api` (or manually: `go 1.23\n\nuse (\n\t./operator\n\t./api\n)`); this lets `api/` import `operator/api/v1alpha1` and `operator/pkg/builder/` without publishing

- [ ] T002 Create `operator/pkg/builder/workflow_builder.go` by moving the `WorkflowBuilder` interface and `DefaultWorkflowBuilder` struct out of `operator/internal/controller/workflow_builder.go`; keep the original file but replace its contents with a package-level re-export (`type WorkflowBuilder = builder.WorkflowBuilder`) so existing controller code compiles without touching every import

- [ ] T003 Update `operator/internal/controller/synthetictest_controller.go` import of `WorkflowBuilder` to reference `operator/pkg/builder/` directly (remove the re-export shim from T002 once all usages are updated); run `go build ./operator/...` to confirm no breakage

- [ ] T004 Create `api/` directory with `go.mod` (`module github.com/downlinklabs/downlink/api`, `go 1.23`); add `chi` and `controller-runtime` dependencies via `go get` from `api/`; create empty skeleton files: `api/cmd/main.go`, `api/internal/httpapi/server.go`, `api/internal/httpapi/store.go`, `api/internal/httpapi/k8s_store.go`, `api/internal/httpapi/handlers/tests.go`, `api/internal/httpapi/handlers/runs.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: CRD status fields, Store interface, server lifecycle wiring. ALL user story work is blocked until this phase is complete.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T005 Add `LastRunPhase string` and `LastRunAt *metav1.Time` fields (with `json:"lastRunPhase,omitempty"` and `json:"lastRunAt,omitempty"` tags) to `SyntheticTestStatus` in `operator/api/v1alpha1/synthetictest_types.go`; run `make generate manifests` from `operator/` to regenerate `zz_generated.deepcopy.go` and CRD manifests

- [ ] T006 Update `SyntheticTestRunReconciler.Reconcile` in `operator/internal/controller/synthetictestrun_controller.go` to patch the parent `SyntheticTest` status after each phase transition: set `Status.LastRunPhase = run.Status.Phase` and `Status.LastRunAt` to `run.Status.FinishedAt` (or `run.Spec.StartedAt` if still Running); use `r.Status().Patch()` to avoid overwriting unrelated status fields

- [ ] T007 [P] Define view types (`SyntheticTestView`, `SyntheticTestRunView`, `TriggerResult`, `ErrorResponse`) and the `Store` interface in `api/internal/httpapi/store.go`:
  ```go
  type Store interface {
      ListTests(ctx context.Context, clusterID string) ([]SyntheticTestView, error)
      GetTest(ctx context.Context, clusterID, namespace, name string) (*SyntheticTestView, error)
      ListRuns(ctx context.Context, clusterID, namespace, testName string, limit int) ([]SyntheticTestRunView, error)
      GetRun(ctx context.Context, clusterID, namespace, name string) (*SyntheticTestRunView, error)
  }
  ```
  Also define `ErrNotFound` sentinel error. Add a comment explaining that `clusterID` is the multi-cluster seam per FR-007/FR-008.

- [ ] T008 [P] Register a field index on `SyntheticTestRun` spec field `spec.name` in `api/cmd/main.go` via `mgr.GetFieldIndexer().IndexField(ctx, &gridv1alpha1.SyntheticTestRun{}, "spec.name", ...)` before `mgr.Start()`; this enables `ListRuns` to use `client.MatchingFields{"spec.name": testName}` from the cache

- [ ] T009 Implement `Server` struct and `NewServer` constructor in `api/internal/httpapi/server.go`; struct holds `addr string`, `apiKey string`, `clusterID string`, `store Store`, `client client.Client`, `builder builder.WorkflowBuilder`; implement `Start(ctx context.Context) error` (builds chi router, starts `http.Server`, shuts down gracefully on ctx cancellation) and `NeedLeaderElection() bool` (returns false) satisfying `manager.Runnable`

- [ ] T010 Implement auth middleware in `api/internal/httpapi/server.go`; validates `Authorization: Bearer <key>` header against `s.apiKey`; on failure returns `401` with body `{"error":"unauthorized","code":"unauthorized"}`; applies to all routes under `/api/v1`

- [ ] T011 Implement `/healthz` and `/readyz` route handlers (no auth; return `200` `ok`) in `api/internal/httpapi/server.go`

- [ ] T012 Implement `api/cmd/main.go`: parse `--api-key string`, `--cluster-id string` (default `"default"`), `--bind-address string` (default `":8082"`) flags; create a controller-runtime manager with the operator's scheme (register `gridv1alpha1` and `wfv1` types) but no reconcilers; construct `httpapi.NewServer(...)` and register with `mgr.Add(server)`; call `mgr.Start(ctrl.SetupSignalHandler())`

**Checkpoint**: `go build ./api/...` succeeds; `api` binary starts, `/healthz` returns 200, `/api/v1/tests` returns 401 without a key.

---

## Phase 3: User Story 1 — Browse SyntheticTests and Run History (Priority: P1) 🎯 MVP

**Goal**: Callers can list all SyntheticTests, fetch a single test, list its run history, and fetch a single run — all backed by the controller-runtime cache.

**Independent Test**: Run quickstart.md Scenarios 1, 2, 4, 5, 6 against a live cluster.

- [ ] T013 [US1] Implement `k8sStore` struct (holds `client.Client` and `clusterID string`) in `api/internal/httpapi/k8s_store.go`; add `NewK8sStore(client, clusterID)` constructor; implement `ListTests`: list all `SyntheticTest` CRs across namespaces using the cache-backed client, project each to `SyntheticTestView` mapping `Status.LastRunPhase` → `LastRunPhase` (null if empty) and `Status.LastRunAt` → `LastRunAt` (null if zero)

- [ ] T014 [P] [US1] Implement `k8sStore.GetTest` in `api/internal/httpapi/k8s_store.go`; fetch by namespace/name; return `ErrNotFound` when `client.IgnoreNotFound(err) == nil`; project to `SyntheticTestView`

- [ ] T015 [US1] Implement `k8sStore.ListRuns` in `api/internal/httpapi/k8s_store.go`; list `SyntheticTestRun` CRs using `client.MatchingFields{"spec.name": testName}` in the given namespace (requires T008 field index); clamp `limit` to [1, 100] defaulting to 20; sort by `Spec.StartedAt` descending (nil StartedAt sorts last); project each to `SyntheticTestRunView` with `ArtifactURL` as null string when `Status.ArtifactURL == ""`, `Message` as null when `Status.Message == ""`

- [ ] T016 [P] [US1] Implement `k8sStore.GetRun` in `api/internal/httpapi/k8s_store.go`; fetch by namespace/name; return `ErrNotFound` when not present; project to `SyntheticTestRunView`

- [ ] T017 [P] [US1] Implement `GET /api/v1/tests` handler (returns `{"tests":[...]}`) and `GET /api/v1/tests/{namespace}/{name}` handler (returns single test object; maps `ErrNotFound` to `404 {"error":"SyntheticTest {ns}/{name} not found","code":"not_found"}`) in `api/internal/httpapi/handlers/tests.go`

- [ ] T018 [US1] Implement `GET /api/v1/tests/{namespace}/{name}/runs` handler in `api/internal/httpapi/handlers/runs.go`; parse `limit` query parameter as integer (default 20, max 100); verify parent SyntheticTest exists via `store.GetTest` first (404 if not); return `{"runs":[...]}`

- [ ] T019 [P] [US1] Implement `GET /api/v1/runs/{namespace}/{name}` handler in `api/internal/httpapi/handlers/runs.go`; maps `ErrNotFound` to `404 {"error":"SyntheticTestRun {ns}/{name} not found","code":"not_found"}`

- [ ] T020 [US1] Register US1 routes under the auth middleware in the chi router in `api/internal/httpapi/server.go`:
  - `GET /api/v1/tests` → `handlers.ListTests`
  - `GET /api/v1/tests/{namespace}/{name}` → `handlers.GetTest`
  - `GET /api/v1/tests/{namespace}/{name}/runs` → `handlers.ListRuns`
  - `GET /api/v1/runs/{namespace}/{name}` → `handlers.GetRun`

- [ ] T021 [P] [US1] Write handler unit tests for `GET /api/v1/tests` and `GET /api/v1/tests/{namespace}/{name}` in `api/internal/httpapi/handlers/tests_test.go` using `net/http/httptest` and a `fakeStore` stub; cover: list returns all tests with correct fields; get returns 200; get returns 404 on `ErrNotFound`; request without auth header returns 401

- [ ] T022 [P] [US1] Write handler unit tests for `GET /api/v1/tests/{namespace}/{name}/runs` and `GET /api/v1/runs/{namespace}/{name}` in `api/internal/httpapi/handlers/runs_test.go` using `httptest` and `fakeStore`; cover: runs list respects limit; parent not found returns 404; get run returns 404; runs with empty artifactUrl/message serialize as JSON null

**Checkpoint**: `go test ./api/...` passes; quickstart.md Scenarios 1, 2, 4, 5, 6 pass against a live cluster.

---

## Phase 4: User Story 2 — Trigger an On-Demand Run (Priority: P2)

**Goal**: Callers can trigger a named SyntheticTest immediately and poll for the resulting SyntheticTestRun phase transition.

**Independent Test**: Run quickstart.md Scenario 3 (trigger + poll loop) against a live cluster.

- [ ] T023 [US2] Implement `POST /api/v1/tests/{namespace}/{name}/trigger` handler in `api/internal/httpapi/handlers/runs.go`:
  1. Fetch the full `SyntheticTest` CR via `client.Get` (not Store — the builder needs the CRD object); return `404` if not found
  2. Call `store.ListRuns(ctx, clusterID, namespace, name, 100)` and scan for any entry with `Phase == "Running"`; return `409 {"error":"a run for {name} is already in progress","code":"conflict"}` if found
  3. Call `builder.BuildWorkflow(ctx, syntheticTest)` to get an Argo `Workflow` object; append a 5-char lowercase hex suffix to the workflow name to avoid collisions
  4. Create the Workflow via `client.Create(ctx, workflow)`
  5. Create a `SyntheticTestRun` CR: `Spec.Name = testName`, `Spec.WorkflowName = workflow.Name`, `Spec.StartedAt = now`; set owner reference to the SyntheticTest; set label `downlinklabs.io/synthetic-test: testName`
  6. Return `202` with `{"cluster": clusterID, "namespace": namespace, "runName": workflow.Name}`

- [ ] T024 [US2] Register the trigger route under the auth middleware in `api/internal/httpapi/server.go`:
  - `POST /api/v1/tests/{namespace}/{name}/trigger` → `handlers.TriggerRun`

- [ ] T025 [P] [US2] Write handler unit tests for the trigger endpoint in `api/internal/httpapi/handlers/runs_test.go` using `httptest` and a fake `client.Client`; cover: 404 when SyntheticTest not found; 409 when a Running SyntheticTestRun already exists; 202 with correct `runName` on success; created SyntheticTestRun CR has correct `spec.name` and owner reference label

**Checkpoint**: quickstart.md Scenario 3 (trigger + poll) completes successfully.

---

## Phase 5: User Story 3 — Multi-Cluster Identifier (Priority: P3)

**Goal**: Every API response includes a stable `cluster` field sourced from `--cluster-id`, making the response shape ready for future multi-cluster aggregation without a breaking contract change.

**Independent Test**: Deploy with `--cluster-id=staging`; verify all five endpoint responses contain `"cluster":"staging"`.

- [ ] T026 [P] [US3] Write a table-driven unit test in `api/internal/httpapi/handlers/tests_test.go` that constructs a Server with `clusterID = "my-cluster"`, calls all read endpoints via `httptest`, and asserts `response.cluster == "my-cluster"` (or `response.tests[0].cluster`) for each; this confirms `clusterID` flows from flag → Server → Store → view type → JSON response

**Checkpoint**: `--cluster-id` propagates correctly through all five endpoints.

---

## Phase 6: Helm Chart & Polish

**Purpose**: Wire the second binary into the Helm chart and validate end-to-end.

- [ ] T027 [P] Add `api.image`, `api.key`, and `api.clusterID` (default `"default"`) to `charts/downlink/values.yaml`

- [ ] T028 Create `charts/downlink/templates/serviceaccount-api.yaml`: ServiceAccount `downlink-api` in `{{ .Release.Namespace }}`

- [ ] T029 Create `charts/downlink/templates/clusterrole-api.yaml`: ClusterRole granting `get;list;watch` on `synthetictests` and `synthetictestruns` (group `downlinklabs.io`), `get;list;watch;create` on `workflows` (group `argoproj.io`), and `get;list;watch;create;update` on `synthetictestruns` (for trigger write path)

- [ ] T030 [P] Create `charts/downlink/templates/clusterrolebinding-api.yaml` binding the ClusterRole from T029 to the `downlink-api` ServiceAccount

- [ ] T031 Create `charts/downlink/templates/deployment-api.yaml`: Deployment for the API server image with args `--api-key={{ .Values.api.key }}`, `--cluster-id={{ .Values.api.clusterID }}`, `--bind-address=:8082`; liveness probe `GET /healthz :8082`; readiness probe `GET /readyz :8082`; serviceAccountName `downlink-api`

- [ ] T032 [P] Create `charts/downlink/templates/service-api.yaml`: ClusterIP Service `downlink-api` exposing TCP port 8082; add a note in `values.yaml` for port-forwarding the service for local quickstart validation

- [ ] T033 Add `api/Dockerfile` using a multi-stage Go build targeting `api/cmd/main.go`; tag pattern matches the existing operator image convention

- [ ] T034 Run `go vet ./operator/... && go test ./operator/...` to confirm the `WorkflowBuilder` refactor (T002–T003) did not break existing controller tests

- [ ] T035 Run `go vet ./api/... && go test ./api/...` to confirm all handler unit tests pass

- [ ] T036 Run quickstart.md end-to-end validation Scenarios 1–6 against a live cluster with both the operator and API server deployed via the updated Helm chart

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 — BLOCKS all user stories
- **Phase 3 (US1)**: Depends on Phase 2
- **Phase 4 (US2)**: Depends on Phase 2 — can run in parallel with Phase 3 after Phase 2
- **Phase 5 (US3)**: Depends on Phase 3 (needs at least one endpoint to verify cluster field)
- **Phase 6 (Helm/Polish)**: Depends on Phases 3, 4, 5

### Within Phase 1

- T002 (move WorkflowBuilder) must complete before T003 (update controller import)
- T004 (create api/ module) can run in parallel with T002/T003

### Within Phase 2

- T005 must complete before T006 (CRD fields needed before reconciler writes them)
- T007, T008 are independent and can run in parallel with T005/T006

### Parallel Opportunities

```
# Phase 1 parallel:
T002 (move WorkflowBuilder) + T004 (create api/ module skeleton)

# Phase 2 parallel (after T001 go.work):
T005+T006 (CRD fields + reconciler)  alongside  T007+T008 (Store interface + field index)

# Phase 3 parallel start (after Phase 2):
T013 (ListTests) + T014 (GetTest) + T015 (ListRuns) + T016 (GetRun)   ← k8sStore methods
T017 (test handlers) + T018+T019 (runs handlers)                       ← handler files
T021 (test handler tests) + T022 (runs handler tests)                  ← test files

# Phase 4 + Phase 5 can run in parallel after Phase 3

# Phase 6 parallel:
T027 (values.yaml) + T028 (SA) + T029 (ClusterRole) + T033 (Dockerfile) can all start together
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup (T001–T004)
2. Phase 2: Foundational (T005–T012)
3. Phase 3: US1 (T013–T022)
4. **STOP and validate**: Run quickstart.md Scenarios 1, 2, 4, 5, 6
5. This delivers the full read path; the UI can browse tests and history

### Incremental Delivery

1. Phase 1 + Phase 2 → `api` binary starts with health probes and auth
2. Phase 3 (US1) → full read path; run quickstart.md validation
3. Phase 4 (US2) → trigger endpoint; run quickstart.md Scenario 3
4. Phase 5 (US3) → cluster ID tests; verify multi-cluster readiness
5. Phase 6 → Helm chart wired; full chart deployment validated

---

## Notes

- `SyntheticTestStatus.LastRunPhase` and `LastRunAt` do not exist yet — T005 is a prerequisite for the read model to return non-null values
- `SyntheticTestRun` CRs are matched to their parent SyntheticTest via `Spec.Name`; T008 sets up the field index required for efficient cache-backed list queries in T015
- The trigger handler (T023) holds the `client.Client` directly (not via Store) because it creates Kubernetes resources; Store is read-only by design per research.md
- `WorkflowBuilder` moves from `operator/internal/controller/` to `operator/pkg/builder/` specifically to clear Go's `internal/` restriction so `api/` can import it
- The API server needs NO leader election — it is stateless and multiple replicas can safely serve read traffic simultaneously; only the trigger path requires care (the 409 conflict check handles concurrent triggers)
