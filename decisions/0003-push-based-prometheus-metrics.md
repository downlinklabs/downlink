# 3. Push-based Prometheus metrics via Pushgateway

Date: 2026-10-21

## Status

Accepted

## Context

Test runner pods are ephemeral — they start, run Playwright, and exit. Prometheus's default pull model
requires a running HTTP endpoint to scrape; a pod that exits in seconds is never scraped. We needed a way
to get test results into Prometheus without keeping pods alive.

## Decision

We use the Prometheus Pushgateway. Test runners push metrics (`synthetic_step_duration_seconds`,
`synthetic_test_duration_seconds`) to the Pushgateway immediately after a run. Prometheus scrapes the Pushgateway
on its normal interval. The Pushgateway runs as a persistent service in the cluster.

## Consequences

- Pushgateway is a cluster dependency alongside Prometheus.
- Push semantics are less reliable than pull: a pod crash before push means the run is invisible to Prometheus.
- `pushAdd` (not `push`) must be used so parallel workers accumulate observations rather than overwrite each other.
- Pushgateway retains the last pushed value; stale metrics are possible if a test stops running without cleanup.
