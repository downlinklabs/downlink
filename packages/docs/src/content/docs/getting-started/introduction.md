---
title: Introduction
description: What downlink is and why you'd use it.
---

downlink brings synthetic testing closer to your infrastructure. Write Playwright tests for your critical user paths, run them inside your Kubernetes cluster on a schedule or on demand, and pull the results into your existing observability stack.

## What it does

- **Run Playwright tests in-cluster** — no external SaaS dependency, your tests run where your app lives.
- **Flexible triggers** — POSIX cron schedule, on-demand via API, or event-driven from your CI pipeline.
- **Metrics you own** — results flow to any Prometheus-compatible or OTLP backend.
- **SLO-ready** — define service level objectives directly on critical user path metrics.
- **Failure artifacts** — screenshots and Playwright trace view on every failure, automatically stored.
- **Test metadata** — annotate tests with service names, teams, environments, and more.

## How it works

1. You write standard Playwright tests using `@downlink/synthetics`.
2. The downlink operator runs in your Kubernetes cluster and schedules test execution.
3. The runner executes tests and emits metrics and artifacts.
4. You query results through your existing monitoring tools.

## Next steps

- [Install downlink](/getting-started/installation/) in your cluster
- [Run your first synthetic test](/getting-started/quick-start/)
