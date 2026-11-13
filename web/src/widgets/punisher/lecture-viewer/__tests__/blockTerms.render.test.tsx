import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LectureBlock } from "@/src/shared/api/punisher";
import { BlockRenderer } from "../ui/blocks/BlockRenderer";

// 용어 팝오버는 서버 조회가 있으므로 표지만 남긴다 — 여기서는 연결 여부만 본다.
vi.mock("@/src/features/punisher/term-popover", () => ({
  TermPopover: ({ term, conceptId }: { term: string; conceptId: string }) => <button type="button" data-concept={conceptId}>{term}</button>,
}));

const BODIED: LectureBlock[] = [
  { anchor: "m", type: "markdown", text: "[해시](term:ds.hash) 본문" },
  { anchor: "k", type: "keypoint", kind: "rule", text: "[캐시](term:arch.cache) 규칙" },
  { anchor: "e", type: "exercise", text: "[스택](term:ds.stack) 문제" },
  { anchor: "s", type: "summary", text: "- [힙](term:ds.heap) 요약" },
];

describe("본문 블록의 용어 링크", () => {
  it("opens the term popover from keypoint · exercise · summary like a markdown block", () => {
    for (const block of BODIED) {
      render(<BlockRenderer block={block} onBeforeDeeper={() => undefined} />);
    }
    for (const [term, id] of [["해시", "ds.hash"], ["캐시", "arch.cache"], ["스택", "ds.stack"], ["힙", "ds.heap"]]) {
      expect(screen.getByRole("button", { name: term })).toHaveAttribute("data-concept", id);
    }
  });
});
