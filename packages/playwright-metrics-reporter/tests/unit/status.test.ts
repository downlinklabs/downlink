import { describe, it, expect } from "vitest";
import type { TestResult, TestStep } from "@playwright/test/reporter";
import {
  deriveTestStatus,
  deriveStepStatus,
} from "../../src/reporter/status.js";

function makeResult(status: TestResult["status"]): TestResult {
  return {
    status,
    duration: 100,
    errors: [],
    steps: [],
    stderr: [],
    stdout: [],
    attachments: [],
    annotations: [],
    parallelIndex: 0,
    retry: 0,
    workerIndex: 0,
    startTime: new Date(),
  } as unknown as TestResult;
}

function makeStep(error?: { message: string }): TestStep {
  return {
    error,
    duration: 50,
    title: "step",
    category: "test.step",
    steps: [],
    titlePath: () => [],
    annotations: [],
    attachments: [],
  } as unknown as TestStep;
}

describe("deriveTestStatus", () => {
  it("maps passed → success", () =>
    expect(deriveTestStatus(makeResult("passed"))).toBe("success"));
  it("maps failed → failure", () =>
    expect(deriveTestStatus(makeResult("failed"))).toBe("failure"));
  it("maps timedOut → timeout", () =>
    expect(deriveTestStatus(makeResult("timedOut"))).toBe("timeout"));
  it("maps skipped → skip", () =>
    expect(deriveTestStatus(makeResult("skipped"))).toBe("skip"));
  it("maps interrupted → failure", () =>
    expect(deriveTestStatus(makeResult("interrupted"))).toBe("failure"));
});

describe("deriveStepStatus", () => {
  it("returns success when no error", () =>
    expect(deriveStepStatus(makeStep())).toBe("success"));
  it("returns timeout when error message contains Timeout", () =>
    expect(
      deriveStepStatus(makeStep({ message: "Timeout 5000ms exceeded" })),
    ).toBe("timeout"));
  it("returns timeout case-insensitively", () =>
    expect(
      deriveStepStatus(makeStep({ message: "timeout waiting for selector" })),
    ).toBe("timeout"));
  it("returns failure for assertion errors", () =>
    expect(
      deriveStepStatus(
        makeStep({ message: "expect(received).toBe(expected)" }),
      ),
    ).toBe("failure"));
});
