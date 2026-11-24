---
title: Quick Start
description: Write and run your first downlink synthetic test.
---

This guide walks you through creating a `SyntheticTest` resource with an inline Playwright test.

## Prerequisites

- downlink operator installed (see [Installation](/getting-started/installation/))
- A namespace for your tests

## Create a namespace

```bash
kubectl create namespace my-synthetics
```

## Write your first synthetic test

Create a `SyntheticTest` manifest with your Playwright test written directly in the `source.inline` field:

```yaml
apiVersion: downlinklabs.io/v1alpha1
kind: SyntheticTest
metadata:
  name: my-first-synthetic
  namespace: my-synthetics
spec:
  schedule: "*/5 * * * *"
  entrypoint: run
  container:
    name: playwright
  source:
    inline: |
      import { test, expect } from "@playwright/test";

      test("homepage loads", async ({ page }) => {
        await page.goto("https://your-app.example.com");
        await expect(page.locator("h1")).toBeVisible();
      });
```

Apply it to your cluster:

```bash
kubectl apply -f synthetic-test.yaml
```

## Verify it's running

```bash
kubectl get synthetictests -n my-synthetics
kubectl describe synthetictest my-first-synthetic -n my-synthetics
```

## Using test steps for metrics

Wrap logical actions in `test.step()` calls — downlink emits per-step duration metrics to your Pushgateway, giving you granular visibility into which parts of a flow are slow:

```yaml
source:
  inline: |
    import { test, expect } from "@playwright/test";

    test("checkout flow", async ({ page }) => {
      await page.goto("https://your-app.example.com");

      await test.step("add to cart", async () => {
        await page.getByRole("button", { name: "Add to cart" }).click();
      });

      await test.step("complete checkout", async () => {
        await page.getByRole("link", { name: "Checkout" }).click();
        await expect(page.getByText("Order confirmed")).toBeVisible();
      });
    });
```

## Next steps

- Enable [artifact storage](/getting-started/installation/#enable-artifact-storage) to capture screenshots and traces on failure
