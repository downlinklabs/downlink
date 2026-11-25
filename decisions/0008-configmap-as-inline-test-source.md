# 8. ConfigMap as inline test source

Date: 2026-10-21

## Status

Accepted

## Context

The runner image contains the Playwright runtime (browsers, node_modules) but no test files — test code must be injected
at pod startup. Options considered: bake tests into the image (requires rebuild on every test change), S3 download via init
container (good for CI-published suites, adds init container complexity), or inline in the CR stored as a ConfigMap volume.

## Decision

For the current milestone, test source is embedded in `SyntheticTest.Spec.Source.Inline` as a string. The operator creates
a `<test-name>-config` ConfigMap from this field and mounts it at `/app/tests` in the runner pod. This is the only supported
source mode today.

S3-based source (a key pointing to a versioned test bundle downloaded by an init container) is specified in the product plan
as a future option but is not yet implemented.

## Consequences

- Getting started requires only a `kubectl apply`; no CI pipeline or S3 bucket is needed.
- Test source is stored in etcd as part of the CR; large test suites will hit etcd value size limits (~1 MB).
- Updating a test means editing the CR, which naturally goes through GitOps review.
- The S3 path is already reserved in the spec (`source.s3`) so migrating to it won't break the API.
