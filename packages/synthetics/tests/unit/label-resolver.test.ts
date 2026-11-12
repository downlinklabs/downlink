import { describe, it, expect } from "vitest";
import type { Suite, TestCase } from "@playwright/test/reporter";
import { resolveTestSuite } from "../../src/reporter/label-resolver.js";

function makeSuite(type: Suite["type"], title: string, parent?: Suite): Suite {
  return {
    type,
    title,
    parent,
    suites: [],
    tests: [],
    allTests: () => [],
    entries: () => [],
    project: () => undefined,
    titlePath: () => [],
  } as unknown as Suite;
}

function makeTest(parent: Suite): TestCase {
  return { parent } as unknown as TestCase;
}

describe("resolveTestSuite", () => {
  it("returns outermost describe block title", () => {
    const root = makeSuite("root", "");
    const project = makeSuite("project", "chromium", root);
    const file = makeSuite("file", "/app/tests/http-bin.spec.ts", project);
    const describe = makeSuite("describe", "HTTP Methods", file);
    const test = makeTest(describe);
    expect(resolveTestSuite(test)).toBe("HTTP Methods");
  });

  it("returns spec basename when no describe block", () => {
    const root = makeSuite("root", "");
    const project = makeSuite("project", "chromium", root);
    const file = makeSuite("file", "/app/tests/http-bin.spec.ts", project);
    const test = makeTest(file);
    expect(resolveTestSuite(test)).toBe("http-bin.spec.ts");
  });

  it("returns outermost describe when nested", () => {
    const root = makeSuite("root", "");
    const project = makeSuite("project", "chromium", root);
    const file = makeSuite("file", "/app/tests/auth.spec.ts", project);
    const outer = makeSuite("describe", "Auth", file);
    const inner = makeSuite("describe", "Login", outer);
    const test = makeTest(inner);
    // walks up to first describe found — inner is first parent
    expect(resolveTestSuite(test)).toBe("Login");
  });
});
