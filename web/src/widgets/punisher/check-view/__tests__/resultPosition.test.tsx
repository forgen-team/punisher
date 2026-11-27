import { fireEvent, render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { FeedbackProvider, UIProvider } from "@forgen-team/ui";
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
  useCheckCancel: () => idle,
  useCheckResume: () => idle,
  useCheckDispute: () => idle,
  useToday: () => ({ data: { unchecked: [] } }),
  useStartCheck: () => idle,
  useRelearn: () => idle,
  useAcknowledgeMisconception: () => idle,
  useCompleteLecture: () => idle,
  useStartReview: () => idle,
}));

vi.mock("../../lecture-viewer", () => ({
  BlockRenderer: ({ block }: { block: { text: string } }) => <p>{block.text}</p>,
}));

const { CheckView } = await import("../CheckView");

// 그만두기 확인 창·되돌리기 알림(useConfirm·useToast)이 앱 뿌리의 Provider 를 쓴다.
const wrapper = ({ children }: { children: ReactNode }) => (
  <UIProvider>
    <FeedbackProvider>{children}</FeedbackProvider>
  </UIProvider>
);

describe("끝난 복습 세션", () => {
  it("결과 요약이 문항 기록보다 위에 있고, 아래 다음 행동 '오늘로'가 있다", () => {
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    render(<CheckView checkId="s1" />, { wrapper });
    const result = screen.getByRole("heading", { name: "결과" });
    const item = screen.getByText("페이지 캐시는 어디에 있나요?");
    expect(result.compareDocumentPosition(item) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByRole("link", { name: "오늘로" }).getAttribute("href")).toBe("/");
    expect(screen.getByText("복습 1개를 마쳤어요")).toBeInTheDocument();
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
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.getByText("인정함").tagName).toBe("SPAN");
    expect(screen.queryByRole("button", { name: /인정/ })).toBeNull();
    expect(screen.queryByText(/처음 예측/)).toBeNull();
  });

  it("'이 강의 한 장 요약'은 접혀 있다가 펼치면 보인다 — 확인은 결과 아래 한 번, 복습은 문항 아래, 요약이 없으면 없다", () => {
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    const withSummary = { ...base.items[0], lectureSummary: "- 읽기는 메모리에서" };
    finished = { ...base, session: { ...base.session, kind: "check", lectureId: "l1" }, items: [withSummary] };
    const { unmount } = render(<CheckView checkId="s1" />, { wrapper });
    const result = screen.getByRole("heading", { name: "결과" }).closest("section")!;
    const toggle = within(result).getByRole("button", { name: "이 강의 한 장 요약" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("읽기는 메모리에서")).toBeNull();
    fireEvent.click(toggle);
    expect(screen.getByText("읽기는 메모리에서")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "이 강의 한 장 요약" })).toHaveLength(1);
    unmount();

    finished = { ...base, items: [withSummary] };
    const review = render(<CheckView checkId="s1" />, { wrapper });
    const item = screen.getByRole("region", { name: "문항 1/1" });
    expect(within(item).getByRole("button", { name: "이 강의 한 장 요약" })).toBeInTheDocument();
    review.unmount();

    finished = base;
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.queryByRole("button", { name: "이 강의 한 장 요약" })).toBeNull();
  });
});
