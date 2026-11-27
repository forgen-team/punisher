import { describe, expect, it } from "vitest";
import type { StudyRequest } from "@/src/shared/api/punisher";
import { splitStudyRequests, studyRequestMeta } from "../ui/StudyRequestList";

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

describe("studyRequestMeta", () => {
  const names = new Map([["c1", "TCP 연결"]]);
  it("RSS 에서 온 요청은 meta 앞에 'RSS 글에서', 준비됨이면 주제 개념 이름", () => {
    expect(studyRequestMeta(req("a", {}), names)).toBe("RSS 글에서 · TCP 연결");
    expect(studyRequestMeta(req("b", { source: "manual" }), names)).toBe("TCP 연결");
  });

  it("찾는 중·실패는 진행 문구", () => {
    expect(studyRequestMeta(req("c", { source: "manual", status: "unresolved", resolving: true }), names)).toBe("개념 찾는 중 · 보통 1분 안에 끝나요");
    expect(studyRequestMeta(req("d", { source: "manual", status: "unresolved", resolveAttempts: 2, resolveError: "no matching concept" }), names)).toBe("지도에 맞는 개념이 없어요");
  });
});
