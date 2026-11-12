import type { TestResult, TestStep } from "@playwright/test/reporter";

export type TestStatus = "success" | "failure" | "timeout" | "skip";
export type StepStatus = "success" | "failure" | "timeout";

export function deriveTestStatus(result: TestResult): TestStatus {
  switch (result.status) {
    case "passed":
      return "success";
    case "timedOut":
      return "timeout";
    case "skipped":
      return "skip";
    case "failed":
    case "interrupted":
    default:
      return "failure";
  }
}

export function deriveStepStatus(step: TestStep): StepStatus {
  if (!step.error) return "success";
  if (/timeout/i.test(step.error.message ?? "")) return "timeout";
  return "failure";
}
