import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { FeedbackProvider, UIProvider } from "@forgen-team/ui";
import type { CheckDetail } from "@/src/shared/api/punisher";
import type { PageAction } from "@/src/shared/lib/screen-chrome";

const idle = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false, isSuccess: false, error: null, variables: undefined };
const abandon = { ...idle, mutate: vi.fn() };
const startCheck = { ...idle, mutate: vi.fn() };
let cancel: Omit<typeof idle, "error"> & { error: unknown } = { ...idle, mutate: vi.fn() };
const replace = vi.fn();
let pageActions: PageAction[] | null = null;
let titleSet: string | null | undefined;
let today: { unchecked: { lectureId: string }[] } = { unchecked: [] };

const review: CheckDetail = {
  session: {
    id: "s1", kind: "review", grader: "llm", conceptId: "", lectureId: "", itemIds: ["i1", "i2", "i3"], cursor: 1,
    status: "active", result: null, createdAt: "", updatedAt: "", finishedAt: null,
  },
  items: [
    { id: "i1", objectiveId: "o1", format: "recall", prompt: "첫 질문", done: true, outcome: "unaided_correct", answer: "첫 모범답" },
    { id: "i2", objectiveId: "o2", format: "recall", prompt: "둘째 질문", done: false },
    { id: "i3", objectiveId: "o3", format: "recall", prompt: "셋째 질문", done: false },
  ],
  turns: [],
  current: { itemId: "i2", answers: 0, canAnswer: true, canReveal: true, canSelfGrade: false },
  pending: false,
  mastery: [],
};

let detail: CheckDetail = review;

vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => ({ push: vi.fn(), replace, back: vi.fn() }),
  Link: ({ href, children }: { href: string; children?: ReactNode }) => <a href={href}>{children}</a>,
}));

vi.mock("@/src/shared/lib/screen-chrome", () => ({
  usePageActions: (actions: PageAction[] | null) => { pageActions = actions; },
  useScreenTitle: (title: string | null | undefined) => { titleSet = title; },
}));

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useCheckSession: () => ({ data: detail, error: null, refetch: vi.fn(), isRefetching: false }),
  useLecture: () => ({ data: undefined }),
  useCheckRetry: () => idle,
  useCheckAbandon: () => abandon,
  useCheckCancel: () => cancel,
  useCheckResume: () => idle,
  useCheckDispute: () => idle,
  useCheckAnswer: () => idle,
  useCheckReveal: () => idle,
  useCheckSelfGrade: () => idle,
  useToday: () => ({ data: today }),
  useStartCheck: () => startCheck,
  useRelearn: () => idle,
  useAcknowledgeMisconception: () => idle,
  useCompleteLecture: () => idle,
}));

vi.mock("../../lecture-viewer", () => ({
  BlockRenderer: ({ block }: { block: { text: string } }) => <p>{block.text}</p>,
}));

const { CheckView } = await import("../CheckView");

const wrapper = ({ children }: { children: ReactNode }) => (
  <UIProvider>
    <FeedbackProvider>{children}</FeedbackProvider>
  </UIProvider>
);

describe("진행 중 복습 세션", () => {
  it("제목은 종류 이름, 진행 줄 '2/3 · 복습'과 Footer [그냥 알려줘][보내기]", () => {
    detail = review;
    render(<CheckView checkId="s1" />, { wrapper });
    expect(titleSet).toBe("복습");
    expect(screen.getByText("2/3 · 복습")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "33");
    expect(screen.getByText("둘째 질문")).toBeInTheDocument();
    expect(screen.queryByText("첫 질문")).toBeNull();
    expect(screen.getByRole("button", { name: "그냥 알려줘" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "보내기" })).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox", { name: "답" }), { target: { value: "내 답" } });
    expect(screen.getByRole("button", { name: "보내기" })).toBeEnabled();
  });

  it("본문에는 글자 메뉴(그만두기·건너뛰기)가 없고 그만두기는 페이지 동작 ⋯ 의 위험 항목", () => {
    detail = review;
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.queryByRole("button", { name: "그만두기" })).toBeNull();
    expect(pageActions).toEqual([expect.objectContaining({ id: "stop", label: "그만두기", danger: true })]);
  });

  it("그만두기는 확인 창을 먼저 — 취소하면 실행하지 않고, 그만두기를 누르면 세션을 닫는다", async () => {
    detail = review;
    abandon.mutate.mockClear();
    render(<CheckView checkId="s1" />, { wrapper });
    pageActions?.[0]?.onSelect?.();
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("'복습'을 여기서 그만둘까요?");
    expect(dialog).toHaveTextContent("답한 1개는 저장하고, 남은 2개는 내일 복습으로 넘겨요.");
    fireEvent.click(screen.getByRole("button", { name: "취소" }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    expect(abandon.mutate).not.toHaveBeenCalled();

    pageActions?.[0]?.onSelect?.();
    await screen.findByRole("alertdialog");
    fireEvent.click(screen.getByRole("button", { name: "그만두기" }));
    await waitFor(() => expect(abandon.mutate).toHaveBeenCalledTimes(1));
  });

  it("강의가 있는 세션이면 ⋯ 에 '강의에서 보기'", () => {
    detail = { ...review, session: { ...review.session, kind: "check", lectureId: "l1" } };
    render(<CheckView checkId="s1" />, { wrapper });
    expect(titleSet).toBe("확인하기");
    expect(pageActions?.[0]).toMatchObject({ label: "강의에서 보기", href: "/lectures/l1" });
    expect(pageActions?.[1]).toMatchObject({ danger: true });
  });
});

describe("문항을 끝낸 뒤", () => {
  it("방금 끝낸 문항의 모범답을 보이고 [다음 문항]을 누르면 다음 문항으로", () => {
    detail = review;
    const { rerender } = render(<CheckView checkId="s1" />, { wrapper });
    detail = {
      ...review,
      session: { ...review.session, cursor: 2 },
      items: [review.items[0]!, { ...review.items[1]!, done: true, outcome: "unaided_correct", answer: "둘째 모범답" }, review.items[2]!],
      current: { itemId: "i3", answers: 0, canAnswer: true, canReveal: true, canSelfGrade: false },
    };
    rerender(<CheckView checkId="s1" />);
    expect(screen.getByText("둘째 모범답")).toBeInTheDocument();
    expect(screen.getByText("2/3 · 복습")).toBeInTheDocument();
    expect(screen.queryByText("셋째 질문")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "다음 문항" }));
    expect(screen.getByText("셋째 질문")).toBeInTheDocument();
    expect(screen.getByText("3/3 · 복습")).toBeInTheDocument();
  });

  it("마지막 문항은 [결과 보기] → 끝 화면", () => {
    detail = { ...review, session: { ...review.session, cursor: 2 }, items: [review.items[0]!, review.items[1]!, { ...review.items[2]!, done: true, outcome: "unaided_correct", answer: "셋째 모범답" }], current: { itemId: "i3", answers: 0, canAnswer: true, canReveal: true, canSelfGrade: false } };
    const { rerender } = render(<CheckView checkId="s1" />, { wrapper });
    detail = {
      ...detail,
      session: { ...detail.session, status: "finished", cursor: 3, result: { items: [{ itemId: "i1", objectiveId: "o1", outcome: "unaided_correct", levelBefore: 1, levelAfter: 2, missing: [], anchor: "", misconception: "", nextReviewAt: "2026-12-02T00:00:00+09:00", lapses: 0 }] } },
      current: null,
    };
    rerender(<CheckView checkId="s1" />);
    fireEvent.click(screen.getByRole("button", { name: "결과 보기" }));
    expect(screen.getByText("복습 3개를 마쳤어요")).toBeInTheDocument();
    expect(screen.getByText("다음 복습 12월 2일")).toBeInTheDocument();
  });
});

describe("복습 끝 화면의 다음 행동 1개", () => {
  const finished: CheckDetail = {
    ...review,
    session: { ...review.session, status: "finished", cursor: 3, result: { items: [] } },
    items: review.items.map((it) => ({ ...it, done: true })),
    current: null,
  };

  it("확인 대기가 없으면 [오늘로] 하나", () => {
    detail = finished;
    today = { unchecked: [] };
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.getByRole("link", { name: "오늘로" }).getAttribute("href")).toBe("/");
    expect(screen.queryByRole("button", { name: /확인 대기/ })).toBeNull();
  });

  it("확인 대기가 남았으면 [확인 대기 n개 이어서] — 첫 확인 세션을 시작해 replace", () => {
    detail = finished;
    today = { unchecked: [{ lectureId: "l7" }, { lectureId: "l8" }] };
    startCheck.mutate.mockImplementation((_id: string, options: { onSuccess: (d: { session: { id: string } }) => void }) => options.onSuccess({ session: { id: "c9" } }));
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.queryByRole("link", { name: "오늘로" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "확인 대기 2개 이어서" }));
    expect(startCheck.mutate).toHaveBeenCalledWith("l7", expect.anything());
    expect(replace).toHaveBeenCalledWith("/check/c9");
  });
});

describe("문항 만드는 중 [취소]", () => {
  const preparing: CheckDetail = {
    ...review,
    session: { ...review.session, kind: "check", lectureId: "L1", status: "preparing", cursor: 0, itemIds: [] },
    items: [], current: null, pending: true, generationJobId: "job-1",
  };

  it("[취소]를 누르면 확인 창 없이 바로 취소하고 오늘로 replace + 토스트 '문제 만들기를 취소했어요'", async () => {
    detail = preparing;
    replace.mockClear();
    cancel = { ...idle, error: null, mutate: vi.fn((_: unknown, opts?: { onSuccess?: () => void }) => opts?.onSuccess?.()) };
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.getByText("문항을 만드는 중이에요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "취소" }));
    expect(cancel.mutate).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(replace).toHaveBeenCalledWith("/");
    expect(await screen.findByText("문제 만들기를 취소했어요")).toBeInTheDocument();
  });

  it("취소가 실패하면 그 자리에 인라인 오류(409 는 '이미 끝난 작업'), 이동하지 않는다", async () => {
    const { PunisherApiError } = await import("@/src/shared/api/punisher");
    detail = preparing;
    replace.mockClear();
    cancel = { ...idle, mutate: vi.fn(), error: new Error("network") };
    const { unmount } = render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.getByRole("alert")).toHaveTextContent("취소하지 못했어요. 다시 시도해 주세요");
    unmount();
    cancel = { ...idle, mutate: vi.fn(), error: new PunisherApiError(409, "CHECK_STATE", "지금은 할 수 없는 동작이에요") };
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.getByRole("alert")).toHaveTextContent("이미 끝난 작업이라 취소할 수 없어요");
    expect(replace).not.toHaveBeenCalled();
  });

  it("생성 작업이 없는(진단 등) 준비 화면에는 [취소]가 없다", () => {
    detail = { ...preparing, generationJobId: undefined };
    cancel = { ...idle, error: null, mutate: vi.fn() };
    render(<CheckView checkId="s1" />, { wrapper });
    expect(screen.queryByRole("button", { name: "취소" })).toBeNull();
  });
});
