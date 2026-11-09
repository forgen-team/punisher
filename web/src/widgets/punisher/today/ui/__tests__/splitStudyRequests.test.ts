import { describe, expect, it } from "vitest";
import type { StudyRequest } from "@/src/shared/api/punisher";
import { splitStudyRequests } from "../StudyRequestList";

function req(id: string, patch: Partial<StudyRequest>): StudyRequest {
  return { id, query: id, source: "rss", status: "resolved", learned: false, resolvedConceptIds: ["c1"], ...patch } as StudyRequest;
}

describe("splitStudyRequests", () => {
  it("다 배운 요청은 완료로, 목록에서 뺀 요청은 숨긴다", () => {
    const { active, done } = splitStudyRequests([
      req("a", {}),
      req("b", { learned: true }),
      req("c", { status: "dismissed" }),
      req("d", { status: "dismissed", learned: true }),
      req("e", { status: "unresolved", resolvedConceptIds: [] }),
    ]);
    expect(active.map((r) => r.id)).toEqual(["a", "e"]);
    expect(done.map((r) => r.id)).toEqual(["b"]);
  });
});
