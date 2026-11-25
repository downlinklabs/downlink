# 7. Argo native artifact storage for HTML reports

Date: 2026-10-21

## Status

Accepted

## Context

Playwright produces an HTML report (with embedded screenshots and traces) at `/app/playwright-report` after each run.
We needed to get this into durable storage. Options: a custom upload script added as a workflow step, a sidecar container,
or Argo's built-in artifact collection mechanism.

## Decision

We use Argo's native `ArtifactRepositoryRef` mechanism. The operator reconciles a `<test-name>-artifact-repository`
ConfigMap containing S3 configuration (bucket, endpoint, credentials mode, key format). The workflow template declares
`/app/playwright-report` as an output artifact. Argo handles the upload as part of its own lifecycle, after the test container exits.

Credential modes supported:

- `useSDKCreds: true` — covers EC2 instance profile, EKS IRSA (via ServiceAccount annotation), and any ambient AWS credential chain.
- Static secret — access key and secret key referenced from a named Kubernetes Secret.

## Consequences

- No custom upload code to maintain; Argo handles the S3 upload and any retry.
- Works with any S3-compatible store (AWS S3, GCS with interop mode, MinIO) by setting the endpoint.
- Argo's `keyFormat` for artifacts does not support pod label template variables; only workflow-level variables (e.g., `{{workflow.name}}`) are available.
- Upload failure does not fail the test workflow; artifacts are best-effort.
