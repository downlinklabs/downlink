# Research: Cluster API Gateway

## Co-location vs separate service

**Decision**: Separate binary and container image (`api/`), deployed alongside the operator via the same Helm chart.

**Rationale**: Co-locating the HTTP server with the operator creates a security boundary problem (the internet-facing API surface and a cluster-scoped privileged component share a process), couples deployment cadence, and prevents independent scaling of read replicas. The operator uses leader election, so a co-located API would only serve traffic from the elected leader. When the hosted control plane arrives the API will need its own deployment model regardless — separating now avoids a forced refactor then.

The API server runs its own controller-runtime manager (cache only, no reconcilers) with a tighter RBAC scope than the operator: read access to `SyntheticTest` and `SyntheticTestRun`, plus write access to `Workflow` and `SyntheticTestRun` for the trigger path. The manager pattern gives the same graceful shutdown and in-memory cache without inheriting the operator's privileges.

**Alternatives considered**:
- Co-located `Runnable`: simpler for v1 but conflates security domains; forces restart of reconciliation on API deploy; blocks HA read replicas due to leader election
- Separate microservice in a different repo: unnecessary; both binaries share CRD type definitions from the operator module, so a Go workspace (`go.work`) in the monorepo is the right boundary

---

## Router library

**Decision**: [`go-chi/chi`](https://github.com/go-chi/chi) v5

**Rationale**: Lightweight (`net/http`-compatible, zero non-stdlib dependencies beyond the router itself), idiomatic Go, supports URL parameters (`{namespace}`, `{name}`) without reflection magic. Used widely in Go Kubernetes tooling. Standard library `net/http` alone lacks named path parameters cleanly.

**Alternatives considered**:
- `gorilla/mux`: maintenance-mode as of 2022; chi is the idiomatic successor
- `gin`: brings its own context type, requiring more adaptation to `net/http` middleware idioms
- stdlib only: path parameter parsing requires manual string splitting, not worth the boilerplate

---

## Store interface and extensibility

**Decision**: Define a `Store` interface in `operator/internal/httpapi/store.go` with a Kubernetes-backed implementation (`k8sStore`) and a future `CloudStore` placeholder.

**Rationale**: FR-007 requires the data source to be swappable without changing the API contract. The `Store` interface is the seam. For v1, `k8sStore` uses `mgr.GetClient()` (the cache-backed client) which makes reads cheap — no direct API server calls per request. The interface also makes handler unit tests trivial: inject a fake `Store` with `net/http/httptest`, no envtest required.

**Interface shape**:
```go
type Store interface {
    ListTests(ctx context.Context, clusterID string) ([]SyntheticTestView, error)
    GetTest(ctx context.Context, clusterID, namespace, name string) (*SyntheticTestView, error)
    ListRuns(ctx context.Context, clusterID, namespace, testName string) ([]SyntheticTestRunView, error)
    GetRun(ctx context.Context, clusterID, namespace, name string) (*SyntheticTestRunView, error)
}
```

Trigger is handled separately by the existing `WorkflowBuilder` interface (not part of `Store` — different responsibility).

---

## API key auth

**Decision**: Reuse the same API key mechanism established in ADR-0009 for the trigger endpoint. Single `--api-key` flag on the operator binary; validated in an `auth` middleware applied to all `/api/v1/` routes.

**Accepted via**: `Authorization: Bearer <key>` header (standard; works with `curl -H "Authorization: Bearer ..."` and most HTTP client libraries).

**Rationale**: One key per cluster is the right v1 scope. The spec (FR-006) explicitly calls for consistency with the trigger endpoint. No new secret or Helm value needed.

---

## Cluster identifier

**Decision**: For v1 (single cluster), the cluster identifier is configured via a `--cluster-id` flag defaulting to `"default"`. All responses include `"cluster": "<id>"` in every resource.

**Rationale**: FR-008 requires cluster-scoped identifiers from day one so the multi-cluster response shape is stable when the cloud control plane aggregates multiple clusters. Hardcoding now costs nothing; retrofitting later breaks consumers.

---

## Namespace scoping

**Decision**: The API exposes resources across all namespaces the operator has RBAC to read (which is all namespaces, per the existing `ClusterRole`). Namespace is a URL path segment, not a query filter, so callers address resources precisely.

**Rationale**: The operator already has cluster-scoped read access to `SyntheticTest` and `SyntheticTestRun` resources. Restricting the API to a single namespace would require additional configuration with no benefit for v1.
