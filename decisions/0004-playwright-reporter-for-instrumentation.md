# 4. Playwright Reporter plugin for metrics instrumentation

Date: 2026-10-21

## Status

Accepted

## Context

Test runners need to emit Prometheus metrics for every test and step. The naive approach — importing `prom-client`
at the top of each test file — puts boilerplate in every test, creates risk of label drift across files,
and is easy to forget. We considered a custom `test` wrapper function but that requires test authors to import
from a non-standard path and doesn't hook into Playwright's lifecycle events.

## Decision

We implement a Playwright `Reporter` plugin (`@the-grid/synthetics/reporter`) configured once in `playwright.config.ts`.
The reporter subscribes to Playwright's `onTestEnd` and `onStepEnd` lifecycle hooks, emitting histograms automatically.
Test files contain zero instrumentation code.

## Consequences

- Test authors write standard Playwright tests; metrics are free.
- Label consistency is enforced in one place (`LabelResolver`).
- The reporter runs in the Playwright worker process, so metric state is per-worker; `pushAdd` handles parallel accumulation.
- The operator's own `syntheticTestMetrics()` emission (Argo pod runtime) must be removed to avoid a duplicate metric source for `synthetic_test_duration_seconds`.
