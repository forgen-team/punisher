import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ConceptLecture } from "@/src/shared/api/punisher";

const lecture = (over: Partial<ConceptLecture>): ConceptLecture => ({
  id: "l1", title: "페이지 캐시", depth: 1, kind: "library", version: 1, estimatedMinutes: 20, progressState: "in_progress", percent: 40, ...over,
});

let data: ConceptLecture[] = [];

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useConceptLectures: () => ({ data, isLoading: false, error: null, refetch: vi.fn() }),
}));

// 단독판: 라우터 없이 렌더 — Link 는 일반 <a>(witems next/link 와 같이).
vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  Link: ({ href, children, className }: { href: string; children?: ReactNode; className?: string }) => (
    <a href={href} className={className}>{children}</a>
  ),
}));

const { ConceptLecturesSection } = await import("../ui/ConceptLecturesSection");

describe("관련 강의 진행 상태", () => {
  it("shows '다 읽음' for a lecture read to the end but not marked complete, the percent otherwise", () => {
    data = [lecture({ id: "a", title: "끝까지", percent: 100 }), lecture({ id: "b", title: "중간", percent: 40 })];
    render(<ConceptLecturesSection conceptId="os.page-cache" />);
    expect(screen.getByText(/진행 중 40%/)).toBeInTheDocument();
    expect(screen.getByText(/· 다 읽음$/)).toBeInTheDocument();
    expect(screen.queryByText(/100%/)).toBeNull();
  });
});
