import { Registry, Histogram, Pushgateway } from "prom-client";
import type {
  Reporter,
  TestCase,
  TestResult,
  TestStep,
} from "@playwright/test/reporter";
import { resolveTestSuite } from "./label-resolver.js";
import { deriveTestStatus, deriveStepStatus } from "./status.js";

export interface SyntheticsReporterOptions {
  pushgatewayUrl?: string;
  jobName?: string;
}

const STEP_BUCKETS = [0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30];
const TEST_BUCKETS = [1, 5, 10, 30, 60, 120, 300];

export default class SyntheticsReporter implements Reporter {
  private registry: Registry;
  private stepHistogram: Histogram;
  private testHistogram: Histogram;
  private gateway: Pushgateway<"text/plain; version=0.0.4; charset=utf-8"> | null;
  private jobName: string;

  constructor(options: SyntheticsReporterOptions = {}) {
    const url = options.pushgatewayUrl ?? process.env["PUSHGATEWAY_URL"];
    this.jobName = options.jobName ?? "synthetic-tests";

    this.registry = new Registry();

    this.stepHistogram = new Histogram({
      name: "synthetic_step_duration_seconds",
      help: "Duration of individual test steps in seconds",
      labelNames: [
        "test_suite",
        "test_name",
        "step_name",
        "status",
        "environment",
      ],
      buckets: STEP_BUCKETS,
      registers: [this.registry],
    });

    this.testHistogram = new Histogram({
      name: "synthetic_test_duration_seconds",
      help: "End-to-end duration of synthetic test runs in seconds",
      labelNames: ["test_suite", "test_name", "status", "environment"],
      buckets: TEST_BUCKETS,
      registers: [this.registry],
    });

    if (url) {
      this.gateway = new Pushgateway(url, {}, this.registry);
    } else {
      this.gateway = null;
      process.stderr.write(
        "[SyntheticsReporter] no PUSHGATEWAY_URL configured — metrics will not be pushed\n",
      );
    }
  }

  onStepEnd(test: TestCase, _result: TestResult, step: TestStep): void {
    if (step.parent !== undefined || step.category !== "test.step") return;

    this.stepHistogram
      .labels({
        test_suite: resolveTestSuite(test),
        test_name: test.title,
        step_name: step.title,
        status: deriveStepStatus(step),
        environment: process.env["GRID_ENV"] ?? "unknown",
      })
      .observe(step.duration / 1000);
  }

  async onTestEnd(test: TestCase, result: TestResult): Promise<void> {
    this.testHistogram
      .labels({
        test_suite: resolveTestSuite(test),
        test_name: test.title,
        status: deriveTestStatus(result),
        environment: process.env["GRID_ENV"] ?? "unknown",
      })
      .observe(result.duration / 1000);

    if (!this.gateway) return;

    try {
      await this.gateway.pushAdd({ jobName: this.jobName });
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`[SyntheticsReporter] push failed: ${msg}\n`);
    }
  }
}
