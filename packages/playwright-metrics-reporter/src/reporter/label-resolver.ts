import path from "node:path";
import type { TestCase } from "@playwright/test/reporter";

export function resolveTestSuite(test: TestCase): string {
  let suite = test.parent;
  while (suite) {
    if (suite.type === "describe") return suite.title;
    if (suite.type === "file") return path.basename(suite.title);
    suite = suite.parent!;
  }
  return "unknown";
}
