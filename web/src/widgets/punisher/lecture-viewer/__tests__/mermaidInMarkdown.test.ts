import { describe, expect, it } from "vitest";
import { splitMermaid } from "../lib/mermaidInMarkdown";

describe("splitMermaid", () => {
  it("lifts a ```mermaid fence out of markdown", () => {
    expect(splitMermaid("앞 문단\n\n```mermaid\nsequenceDiagram\n  A->>B: 락\n```\n뒤 문단")).toEqual([
      { kind: "markdown", text: "앞 문단\n" },
      { kind: "mermaid", text: "sequenceDiagram\n  A->>B: 락" },
      { kind: "markdown", text: "뒤 문단" },
    ]);
  });

  it("lifts a plain fence and a bare header run, up to the blank line", () => {
    expect(splitMermaid("```\ngraph TD\nA-->B\n```")).toEqual([{ kind: "mermaid", text: "graph TD\nA-->B" }]);
    expect(splitMermaid("설명\nsequenceDiagram\n  participant A as 서버 A\n\n다음 문단")).toEqual([
      { kind: "markdown", text: "설명" },
      { kind: "mermaid", text: "sequenceDiagram\n  participant A as 서버 A" },
      { kind: "markdown", text: "\n다음 문단" },
    ]);
  });

  it("leaves other code and prose mentions alone", () => {
    for (const text of ["```bash\ngraph TD\n```", "sequenceDiagram 은 시간 순서를 그린다", "```go\nfunc main() {}\n```", "pie 차트"]) {
      expect(splitMermaid(text)).toEqual([{ kind: "markdown", text }]);
    }
  });
});
