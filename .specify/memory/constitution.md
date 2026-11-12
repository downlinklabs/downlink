<!--
Sync Impact Report (remove before committing)
=============================================
Version change: 1.0.0 → 1.1.0 (MINOR: Principle V materially expanded; Technology Stack and
Development Workflow updated with new components)
Modified principles:
  - V. Observability by Default — now specifies metric names, label schema, Pushgateway push
    model, and env-var contract for the metrics reporter
Added content:
  - Technology Stack: playwright-runner Docker image, @downlink/synthetics npm package
  - Development Workflow: TypeScript package build and test steps
Removed sections: n/a
Follow-up TODOs: none
-->

# Downlink Constitution

## Core Principles

### I. Kubernetes-Native Design

All configuration is expressed as Kubernetes Custom Resources (CRDs). The operator MUST reconcile
desired state declared in `SyntheticTest` resources into the corresponding cluster objects
(ConfigMaps, ServiceAccounts, RBAC, CronWorkflows). No imperative mutation of cluster state is
permitted outside the reconcile loop. Every resource created by the operator MUST carry an owner
reference back to the originating `SyntheticTest` so garbage collection is handled automatically.

**Rationale**: Declarative CRDs make Downlink composable with any GitOps toolchain (Flux, Argo CD)
and ensure cluster state is always inspectable and reproducible.

### II. Workflow Delegation

Test execution MUST be delegated to Argo Workflows via `CronWorkflow` resources. The operator owns
lifecycle management (CRDs, RBAC, ConfigMaps, artifact repositories) but MUST NOT own execution
logic. Business logic that maps a `SyntheticTest` spec to a `CronWorkflow` spec MUST live in the
`WorkflowBuilder` interface, keeping the reconciler thin and independently testable.

**Rationale**: Separation of scheduling/orchestration from execution lets each layer evolve
independently and keeps the controller surface area small.

### III. Test-First (NON-NEGOTIABLE)

Controller logic MUST be developed test-first using the red-green-refactor cycle. Tests MUST be
written and reviewed before implementation begins. The envtest suite (controller-runtime's
`envtest` package) MUST be used for controller tests; unit tests MUST cover `WorkflowBuilder`
logic without a live cluster. No new reconciler path MAY be merged without a corresponding test
that first fails.

**Rationale**: Controllers manage cluster state; a bug silently drops resources or orphans
workloads. Test-first discipline surfaces contract violations before they reach production.

### IV. GitOps Compatibility

Downlink resources MUST be fully declarable as static YAML manifests. All configuration values
(schedule, container image, artifact storage) MUST be expressible through the `SyntheticTest`
CRD without requiring out-of-band tooling. Helm chart values MUST map 1:1 to CRD fields with no
hidden defaults that differ between chart-driven and raw-manifest installations.

**Rationale**: Users wire Downlink into existing GitOps pipelines; any hidden state or imperative
step breaks reproducibility and drift detection.

### V. Observability by Default

Every `SyntheticTest` MUST emit Prometheus metrics via the `@downlink/synthetics` reporter, which
pushes to a Prometheus Pushgateway after each test completes. Two histograms MUST always be
emitted:

- `synthetic_test_duration_seconds` — labels: `test_suite`, `test_name`, `status`, `environment`
- `synthetic_step_duration_seconds` — labels: `test_suite`, `test_name`, `step_name`, `status`,
  `environment`

The reporter is configured via environment variables: `PUSHGATEWAY_URL` (required; if absent,
metrics are silently dropped to stderr), `GRID_JOB_NAME` (Pushgateway job name, defaults to
`"synthetic-tests"`), and `GRID_ENV` (environment label, defaults to `"unknown"`). These env vars
MUST be injected by the operator into the Argo Workflow pod spec.

When `artifactStorage.enabled` is true, screenshots and Playwright trace files MUST be uploaded to
the configured S3-compatible bucket using the key format `{{workflow.name}}/playwright-report`.
Artifact storage MUST support static credentials, AWS SDK credential chain (IRSA, instance
profile), and GKE Workload Identity without requiring custom operator builds.

**Rationale**: The push model (reporter → Pushgateway) decouples test pods from a live scrape
endpoint, which is necessary for short-lived Argo Workflow pods that complete before a scrape
interval fires.

## Technology Stack

- **Language**: Go (operator); TypeScript/Playwright (test payloads and metrics reporter)
- **Operator framework**: controller-runtime; scaffolded with operator-sdk
- **Execution engine**: Argo Workflows (`CronWorkflow` for scheduling, `Workflow` for on-demand runs)
- **Runner image** (`playwright-runner/`): `node:23-bookworm-slim` base; ships Chromium via
  `npx playwright install chromium`; mounts user test specs at the path set by `TEST_DIR`
- **Metrics reporter** (`packages/playwright-metrics-reporter`, published as `@downlink/synthetics`):
  TypeScript Playwright reporter that pushes histograms to Prometheus Pushgateway via `prom-client`
- **CRDs**: `SyntheticTest` (schedule + source), `SyntheticTestRun` (individual execution record)
- **Packaging**: Helm chart under `charts/downlink`; operator image and runner image published to ECR
- **Metrics backend**: Prometheus Pushgateway (push model); OTLP export supported
- **Artifact storage**: S3-compatible (AWS S3, MinIO, GCS S3 interop)

Dependency upgrades MUST keep `controller-runtime`, `k8s.io/api`, and `k8s.io/apimachinery` at
compatible minor versions. `argo-workflows` client upgrades require a review of `CronWorkflow`
spec changes that could affect existing test schedules. `prom-client` upgrades MUST be validated
against the Pushgateway content-type negotiation (`text/plain; version=0.0.4`).

## Development Workflow

**Operator (Go)**
- **Build & generate**: `make` from `operator/` regenerates CRD manifests and deepcopy code via
  `controller-gen`. Generated files (`zz_generated.deepcopy.go`, CRD YAML) MUST be committed.
- **Unit/integration tests**: `make test` runs the envtest suite. Tests MUST pass before a PR is
  opened.
- **E2E tests**: `make test-e2e` requires a live cluster with Argo Workflows installed. E2E MUST
  pass before a release tag is cut.
- **Linting**: `golangci-lint` configuration lives at `operator/.golangci.yml`. Lint MUST be clean
  on the default ruleset.

**Metrics reporter (TypeScript)**
- **Build**: `npm run build -w @downlink/synthetics` from the workspace root (runs `tsc`). The
  `dist/` output MUST be committed; it is consumed by the runner image at build time.
- **Unit tests**: `npm run test -w @downlink/synthetics` (Vitest). Tests MUST pass before a PR is
  opened.

**Runner image**
- **Build**: `docker build -f playwright-runner/Dockerfile .` from the repo root. The multi-stage
  build compiles `@downlink/synthetics` and installs Playwright Chromium in the runtime layer.
  The runner image tag MUST be pinned in the `SyntheticTest` container spec; floating `latest`
  tags are not permitted in production CRDs.

**Versioning & release**
- Operator image and Helm chart share the `VERSION` in `operator/Makefile` (`MAJOR.MINOR.PATCH`).
  Increment PATCH for bug fixes, MINOR for new CRD fields or features, MAJOR for breaking CRD
  changes requiring migration.
- Runner image and `@downlink/synthetics` package are versioned independently following the same
  `MAJOR.MINOR.PATCH` rules.
- Release: tag `vMAJOR.MINOR.PATCH`; CI builds and pushes operator and runner images to ECR and
  packages the Helm chart.

## Governance

This constitution supersedes all other development conventions for the Downlink project. Amendments
require:

1. A pull request updating `.specify/memory/constitution.md` with the Sync Impact Report
   comment included at the top.
2. At least one reviewer approval from a project maintainer.
3. The Sync Impact Report comment MUST be removed before the PR is merged.

The `CONSTITUTION_VERSION` MUST be incremented following semantic versioning:
- **MAJOR**: removal or redefinition of an existing principle.
- **MINOR**: new principle or section added.
- **PATCH**: clarifications, wording corrections, non-semantic refinements.

All PRs MUST be reviewed for compliance with the principles above. A PR that violates Principle III
(Test-First) MUST NOT be merged regardless of other approvals.

**Version**: 1.1.0 | **Ratified**: 2026-11-11 | **Last Amended**: 2026-11-11
