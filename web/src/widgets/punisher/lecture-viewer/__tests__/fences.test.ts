import { describe, expect, it } from "vitest";
import { closeUnclosedFence } from "../lib/fences";

describe("closeUnclosedFence", () => {
  it("closes a fence the model left open", () => {
    expect(closeUnclosedFence("a\n```mermaid\nflowchart LR\n### 2. 다음\n")).toBe("a\n```mermaid\nflowchart LR\n### 2. 다음\n```");
  });
  it("leaves balanced text alone", () => {
    const t = "a\n```go\nx\n```\nb\n~~~\nc\n~~~";
    expect(closeUnclosedFence(t)).toBe(t);
  });
});
