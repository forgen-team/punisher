import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { CheckDetail, ObjectiveMastery } from "@/src/shared/api/punisher";

const idle = { mutate: vi.fn(), isPending: false, isSuccess: false, error: null, variables: undefined };

const base: CheckDetail = {
  session: {
    id: "s1", kind: "review", grader: "self", conceptId: "", lectureId: "", itemIds: ["i1"], cursor: 1,
    status: "finished", createdAt: "", updatedAt: "", finishedAt: null,
    result: { items: [{ itemId: "i1", objectiveId: "c#1", outcome: "unaided_correct", levelBefore: 1, levelAfter: 3, missing: [], anchor: "", misconception: "", nextReviewAt: null, lapses: 0 }] },
  },
  items: [{ id: "i1", objectiveId: "c#1", format: "recall", prompt: "페이지 캐시는 어디에 있나요?", done: true }],
  turns: [],
  current: null,
  pending: false,
  mastery: [{ objectiveId: "c#1", text: "페이지 캐시 위치", level: 3 } as ObjectiveMastery],
};

let finished: CheckDetail = base;

// 단독판: next/navigation 대신 React Router 래퍼를 바꾼다(라우터 없이 렌더 — Link 는 일반 <a>, next/link 처럼).
vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn() }),
  Link: ({ href, children, className }: { href: string; children?: ReactNode; className?: string }) => (
    <a href={href} className={className}>{children}</a>
  ),
}));

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useCheckSession: () => ({ data: finished, error: null, refetch: vi.fn(), isRefetching: false }),
  useLecture: () => ({ data: undefined }),
  useCheckRetry: () => idle,
  useCheckAbandon: () => idle,
  useCheckDispute: () => idle,
  useReviewSummary: () => ({ data: { due: 0 } }),
  useRelearn: () => idle,
  useAcknowledgeMisconception: () => idle,
  useCompleteLecture: () => idle,
  useStartReview: () => idle,
}));

vi.mock("../../lecture-viewer", () => ({
  LectureTopBar: ({ title }: { title: string }) => <div data-testid="top-bar">{title}</div>,
  LectureLayout: ({ children }: { children: ReactNode }) => <main>{children}</main>,
  BlockRenderer: ({ block }: { block: { text: string } }) => <p>{block.text}</p>,
}));

const { CheckView } = await import("../CheckView");

describe("끝난 복습 세션", () => {
  it("결과 요약이 문항 기록보다 위에 있고, 아래 다음 행동 '오늘로'가 있다", () => {
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    render(<CheckView checkId="s1" />);
    const result = screen.getByRole("heading", { name: "결과" });
    const item = screen.getByText("페이지 캐시는 어디에 있나요?");
    expect(result.compareDocumentPosition(item) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByRole("link", { name: "오늘로" }).getAttribute("href")).toBe("/");
    expect(screen.getByTestId("top-bar").textContent).toBe("복습 · 문항 1/1");
  });

  it("인정한 오개념은 버튼 대신 '인정함' 글자, 지금 답이 없으면 처음 예측 줄을 뺀다", () => {
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    finished = {
      ...base,
      turns: [{ id: "t1", itemId: "i1", seq: 1, role: "tutor", move: "next", text: "", status: "ready", aided: false, reusedItem: false, createdAt: "",
        verdict: "incorrect", grade: { met: [], missing: [], quote: "", misconception: "항상 디스크에서 읽는다", confidence: 0.9 } }],
      misconceptionTurnIds: ["t1"],
      predictions: [{ objectiveId: "c#1", prompt: "어디서 읽을까요?", answer: "디스크", dontKnow: false, at: "" }],
    };
    render(<CheckView checkId="s1" />);
    expect(screen.getByText("인정함").tagName).toBe("SPAN");
    expect(screen.queryByRole("button", { name: /인정/ })).toBeNull();
    expect(screen.queryByText(/처음 예측/)).toBeNull();
  });
});
