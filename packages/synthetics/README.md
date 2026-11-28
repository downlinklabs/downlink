# @downlink/synthetics

Playwright reporter plugin for [Downlink](https://downlinklabs.io) — automatically emits Prometheus metrics for every test and step with zero instrumentation code in your test files.

## Installation

```bash
npm install @downlink/synthetics
```

## Usage

Add the reporter to your `playwright.config.ts`:

```ts
import { defineConfig } from "@playwright/test";

export default defineConfig({
  reporter: [
    ["@downlink/synthetics/reporter"],
  ],
});
```

The reporter reads `PUSHGATEWAY_URL` from the environment. When running inside a Downlink cluster, the operator sets this automatically. For local use, point it at your own Pushgateway:

```bash
PUSHGATEWAY_URL=http://localhost:9091 npx playwright test
```

### Options

```ts
reporter: [
  ["@downlink/synthetics/reporter", {
    pushgatewayUrl: "http://pushgateway.monitoring.svc:9091",
    jobName: "my-app-synthetics",
  }],
]
```

| Option | Environment variable | Default |
|---|---|---|
| `pushgatewayUrl` | `PUSHGATEWAY_URL` | — |
| `jobName` | — | `"synthetic-tests"` |

If neither `pushgatewayUrl` nor `PUSHGATEWAY_URL` is set, the reporter runs but skips the push step — useful for local test runs where you don't have a Pushgateway.

## Metrics

| Metric | Type | Labels |
|---|---|---|
| `synthetic_test_duration_seconds` | Histogram | `test_suite`, `test_name`, `status`, `environment` |
| `synthetic_step_duration_seconds` | Histogram | `test_suite`, `test_name`, `step_name`, `status`, `environment` |

**Labels:**
- `test_suite` — the `describe` block title, or the filename if no `describe` block is used
- `test_name` — the `test()` title
- `step_name` — the `test.step()` title (step metric only)
- `status` — `passed`, `failed`, or `timedOut`
- `environment` — value of `GRID_ENV` environment variable, defaults to `unknown`

## Running with Downlink

When deployed via a `SyntheticTest` CRD, the reporter is pre-configured in the Downlink runner image — no `playwright.config.ts` changes required. See the [Downlink docs](https://docs.downlinklabs.io) for installation and quickstart.
