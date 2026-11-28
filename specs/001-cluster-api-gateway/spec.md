# Feature Specification: Cluster API Gateway

**Feature Branch**: `001-cluster-api-gateway`

**Created**: 2026-11-28

**Status**: Draft

## Purpose

This API exists to decouple the presentation layer (web UI) from the Kubernetes API. The UI must not communicate directly with Kubernetes CRDs. This server is the stable intermediary — abstracting cluster topology, auth, and data shape so the UI can be built against a consistent contract regardless of whether the backing data comes from a local cluster or a future hosted control plane.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse SyntheticTests and Run History (Priority: P1)

A Cluster Operator or Developer opens a web interface to view all synthetic tests running in their cluster and inspect recent run outcomes without using kubectl.

**Why this priority**: This is the core read path the presentation layer must support. Without it, there is no value in building a UI.

**Independent Test**: Can be fully tested by retrieving a list of SyntheticTests and their associated SyntheticTestRuns from the API and verifying the data matches the cluster state.

**Acceptance Scenarios**:

1. **Given** synthetic tests exist in the cluster, **When** a user requests the list of tests, **Then** each test is returned with its name, schedule, and last-run phase and timestamp
2. **Given** a specific SyntheticTest exists, **When** a user requests its run history, **Then** the response includes a list of SyntheticTestRuns ordered by most recent first, each with phase, start time, finish time, and artifact URL if present
3. **Given** a SyntheticTest has never run, **When** a user requests its run history, **Then** an empty list is returned with no error

---

### User Story 2 - Trigger an On-Demand Run (Priority: P2)

A Developer wants to verify their application after a deployment by triggering a synthetic test run immediately from the UI or a CI script, without waiting for the next scheduled execution.

**Why this priority**: Unblocks the CD integration use case. The trigger endpoint is already specified (ADR-0009); this story surfaces it through the presentation API.

**Independent Test**: Can be tested by triggering a named SyntheticTest via the API and observing a new SyntheticTestRun appear with a Running phase.

**Acceptance Scenarios**:

1. **Given** a SyntheticTest exists and is healthy, **When** a user triggers an on-demand run, **Then** a new SyntheticTestRun is created and its identifier is returned
2. **Given** a named SyntheticTest does not exist, **When** a user attempts to trigger it, **Then** a clear not-found error is returned
3. **Given** a run is triggered, **When** the user polls for its status, **Then** the phase transitions from Running to Succeeded or Failed and is reflected in the response

---

### User Story 3 - Multi-Cluster Fleet View (Priority: P3)

An SRE managing multiple Kubernetes clusters wants a single view across all clusters showing which synthetic tests are healthy and which have recently failed, without switching kubeconfig contexts.

**Why this priority**: This is the primary value driver for the future hosted control plane. The API must be designed so this use case can be added without breaking the single-cluster path.

**Independent Test**: Can be tested by registering two cluster connections and verifying that SyntheticTests and SyntheticTestRuns from both appear in a combined response.

**Acceptance Scenarios**:

1. **Given** two clusters are registered, **When** a user requests the fleet view, **Then** tests from both clusters are returned, each identified by their cluster of origin
2. **Given** one cluster is unreachable, **When** a user requests the fleet view, **Then** data from the available cluster is returned and the unreachable cluster is flagged with an error state rather than failing the entire request

---

### Edge Cases

- When the cluster API is temporarily unreachable, the API returns the last-known cached state without error (the controller-runtime cache is the source of truth for read operations)
- When a SyntheticTestRun's referenced Argo Workflow has been garbage-collected, the API returns the SyntheticTestRun CR data as-is — the run record is authoritative and does not depend on the Workflow existing
- When artifact storage is not configured, `artifactUrl` is returned as `null`
- When the API key is invalid or absent, all endpoints return `401 Unauthorized`; namespace-level permission differences are not surfaced (the operator holds a cluster-scoped read role)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The API MUST expose a resource for listing all SyntheticTests within a cluster, including name, namespace, schedule, and last-run summary
- **FR-002**: The API MUST expose a resource for retrieving the run history of a specific SyntheticTest, returning SyntheticTestRuns ordered by start time descending with cursor-based pagination; default page size is 20, maximum is 100; a next-page cursor is returned when more results exist
- **FR-003**: The API MUST expose an action for triggering an on-demand run of a named SyntheticTest and returning the resulting SyntheticTestRun identifier
- **FR-004**: The API MUST expose a resource for retrieving the status of a specific SyntheticTestRun, including phase, timestamps, and artifact URL
- **FR-005**: The API MUST return a stable error response when a requested resource does not exist, distinguishing not-found from server errors
- **FR-006**: The API MUST authenticate all callers using a static API key passed as a request header; the key is read once at startup from an environment variable or flag, stored as a Kubernetes Secret, and consistent with the trigger endpoint auth model; key rotation requires an operator restart
- **FR-007**: The API MUST be designed so that its data source can be swapped from a live Kubernetes cluster to a hosted data store without changing the API contract, enabling the cloud control plane use case; the UI MUST NOT communicate with Kubernetes CRDs directly
- **FR-008**: The API MUST namespace all resources by cluster identifier so responses remain unambiguous in a multi-cluster context from day one
- **FR-009**: Read operations MUST be served from the controller-runtime cache; the API MUST NOT make direct Kubernetes API server calls per request

### Key Entities

- **SyntheticTest** (read model): name, namespace, cluster, schedule, last-run phase, last-run timestamp
- **SyntheticTestRun** (read model): identifier, parent test name, cluster, phase, started-at, finished-at, artifact URL
- **Cluster**: a registered Kubernetes cluster the API is authorised to query; identified by a stable slug

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can retrieve the full list of SyntheticTests and their last-run status in under 2 seconds on a cluster with 100 active tests
- **SC-002**: A triggered on-demand run is reflected as a new SyntheticTestRun within 5 seconds of the API call returning
- **SC-003**: The API contract requires no changes when adding a second cluster data source — consumers do not need to update integration code
- **SC-004**: All read operations remain available and return the last-known cached state when the Kubernetes API server is temporarily unreachable

## Assumptions

- The presentation layer is a Next.js web UI served separately; this spec covers only the server-side API it consumes
- The Next.js frontend calls this API directly from the browser; there is no Next.js API route or Server Action acting as an intermediary
- Transport is HTTP/JSON REST; gRPC was considered and rejected — browsers cannot use native gRPC, and the Connect protocol would add complexity without meaningful benefit for a browser-facing API
- The API server is a separate binary and container image from the operator, deployed as part of the same Helm chart; it has its own ServiceAccount with a tighter RBAC scope than the operator
- For the initial self-hosted deployment, there is exactly one cluster — the multi-cluster identifiers are present but contain only one entry
- SyntheticTest create/update/delete operations remain the responsibility of kubectl and GitOps tooling; this API is read + trigger only for v1
- The API is deployed within the same cluster as the operator for the self-hosted case; network access to the Kubernetes API server is therefore always available
- Artifact URLs are returned as opaque strings; the API does not proxy artifact content
- TLS termination is handled by the ingress/load balancer layer; the API server speaks plain HTTP internally

## Clarifications

### Session 2026-11-28

- Q: When the Kubernetes API is temporarily unreachable, should the API return stale cached data or an explicit error? → A: Return stale cached data (last known state from the controller-runtime cache) with no special error
- Q: Should the run history endpoint paginate results, or return all SyntheticTestRuns in a single response? → A: Cursor-based pagination with configurable limit (default 20, max 100)
- Q: Should the API key be rotatable without restarting the operator, or is a restart acceptable? → A: Restart acceptable — static key read at startup; short-lived credentials add complexity not warranted for v1
- Q: Should the API use gRPC or HTTP/JSON REST as the transport protocol? → A: HTTP/JSON REST — gRPC was considered but rejected; browsers cannot use native gRPC, and the Connect protocol adds complexity without meaningful benefit for a browser-facing Next.js frontend that calls the API directly
