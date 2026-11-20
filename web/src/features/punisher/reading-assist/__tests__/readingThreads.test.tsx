import { useRef } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { NoteInput, Question, ThreadFollowupInput, ThreadStartInput } from "@/src/shared/api/punisher";

const q = (over: Partial<Question>): Question => ({
  id: "x", anchor: "b1", kind: "question", question: "", answer: "", status: "answered",
  linkedConceptIds: null, proposedConcepts: null, createdAt: "2026-11-20T00:00:00Z", quote: "", parentId: "", mode: "question", ...over,
});

let questions: Question[] = [];
let seq = 0;
const api = {
  listQuestions: vi.fn(async () => questions),
  startThread: vi.fn(async (_lectureId: string, input: ThreadStartInput) => {
    const created = q({ id: `root${++seq}`, anchor: input.anchor, quote: input.quote, mode: input.mode, question: input.question ?? "", answer: `answer for ${input.mode}`, createdAt: `2026-11-20T01:0${seq}:00Z` });
    questions = [...questions, created];
    return created;
  }),
  followupThread: vi.fn(async (rootId: string, input: ThreadFollowupInput) => {
    const created = q({ id: `reply${++seq}`, parentId: rootId, mode: input.mode, question: input.question ?? "", answer: `reply answer ${seq}`, createdAt: `2026-11-20T02:0${seq}:00Z` });
    questions = [...questions, created];
    return created;
  }),
  createNote: vi.fn(async (input: NoteInput) => ({ id: "n1", anchor: input.anchor ?? "", kind: input.kind ?? "note", quote: input.quote ?? "", body: "", createdAt: "", updatedAt: "" })),
  startDeepLecture: vi.fn(async () => ({ status: "started" as const, jobId: "j1" })),
};

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return { ...actual, punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }) };
});

// 데스크톱 드래그 선택을 흉내 낸다(jsdom 에는 선택 좌표가 없다).
const selection = { text: "TCP 는 순서를 보장한다", anchor: "b1", rect: { top: 100, bottom: 120, left: 10, right: 200 } };
vi.mock("../useTextSelection", () => ({ useTextSelection: () => selection }));

// jsdom 에는 matchMedia 가 없다 — 고운 포인터 · 좁은 폭(패널 없음 → 시트) 으로 둔다.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({ matches: false, media: query, addEventListener: () => undefined, removeEventListener: () => undefined }),
});

// 읽기 오버레이가 내용 크기를 지켜본다(jsdom 미구현).
globalThis.ResizeObserver = class {
  observe() { return undefined; }
  unobserve() { return undefined; }
  disconnect() { return undefined; }
};

const { ReadingThreadsProvider, useReadingThreads } = await import("../ReadingThreadsProvider");
const { PunisherApiError } = await import("@/src/shared/api/punisher");
const { groupPassages, groupThreads, countPassageThreads, passageQuestionIds } = await import("../lib/threads");
const { ReadingAssist } = await import("../ThreadSurfaces");

function OpenButton({ id, label = "open-thread" }: { id: string; label?: string }) {
  const ctx = useReadingThreads();
  return <button type="button" onClick={() => ctx?.open(id)}>{label}</button>;
}

function Harness({ openId, extraIds = [] }: { openId?: string; extraIds?: string[] }) {
  const ref = useRef<HTMLDivElement>(null);
  return (
    <ReadingThreadsProvider lectureId="lec1" conceptId="net.tcp" getAnchor={() => "fallback"}>
      <div ref={ref}><p>본문</p></div>
      <ReadingAssist containerRef={ref} />
      {openId && <OpenButton id={openId} />}
      {extraIds.map((id) => (
        <OpenButton key={id} id={id} label={`open-${id}`} />
      ))}
    </ReadingThreadsProvider>
  );
}

function renderHarness(openId?: string, extraIds: string[] = []) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness openId={openId} extraIds={extraIds} />
    </QueryClientProvider>,
  );
}

const toolbar = () => screen.getByRole("group", { name: "고른 구절로 할 일" });

describe("desktop selection toolbar", () => {
  beforeEach(() => {
    questions = [];
    seq = 0;
  });

  it.each([
    ["쉽게", "easier"],
    ["깊게", "deeper"],
  ] as const)("%s starts a thread with mode %s and shows the answer", async (label, mode) => {
    renderHarness();
    fireEvent.click(within(toolbar()).getByRole("button", { name: label }));
    await waitFor(() => expect(api.startThread).toHaveBeenCalledWith("lec1", { anchor: "b1", quote: selection.text, mode }));
    expect(await screen.findByText(`answer for ${mode}`)).toBeInTheDocument();
  });

  it("질문 opens an input first and sends the question with mode question", async () => {
    renderHarness();
    fireEvent.click(within(toolbar()).getByRole("button", { name: "질문" }));
    expect(api.startThread).not.toHaveBeenCalled();
    const input = await screen.findByRole("textbox", { name: "이 구절에 대해 질문" });
    fireEvent.change(input, { target: { value: "왜 순서가 중요해?" } });
    fireEvent.click(screen.getByRole("button", { name: "보내기" }));
    await waitFor(() =>
      expect(api.startThread).toHaveBeenCalledWith("lec1", { anchor: "b1", quote: selection.text, mode: "question", question: "왜 순서가 중요해?" }),
    );
  });

  it("메모 saves a highlight note with the quote", async () => {
    renderHarness();
    fireEvent.click(within(toolbar()).getByRole("button", { name: "메모" }));
    await waitFor(() =>
      expect(api.createNote).toHaveBeenCalledWith({ lectureId: "lec1", conceptId: "net.tcp", anchor: "b1", kind: "highlight", quote: selection.text, body: "" }),
    );
    expect(api.startThread).not.toHaveBeenCalled();
  });
});

describe("thread follow-ups", () => {
  beforeEach(() => {
    seq = 10;
    questions = [
      q({ id: "root1", quote: "TCP 는 순서를 보장한다", mode: "easier", question: "server generated sentence", answer: "첫 답" }),
      q({ id: "reply1", parentId: "root1", mode: "question", question: "그럼 UDP 는?", answer: "둘째 답", createdAt: "2026-11-20T00:01:00Z" }),
    ];
  });

  it("appends the follow-up answer and always posts to the root id", async () => {
    renderHarness("root1");
    fireEvent.click(await screen.findByRole("button", { name: "open-thread" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("첫 답")).toBeInTheDocument();
    expect(within(dialog).getByText("둘째 답")).toBeInTheDocument();
    // easier/deeper 루트의 서버 요청 문장은 말풍선으로 보이지 않는다.
    expect(within(dialog).queryByText("server generated sentence")).toBeNull();

    fireEvent.change(within(dialog).getByRole("textbox", { name: "이어서 묻기" }), { target: { value: "예를 들어 줘" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "보내기" }));
    await waitFor(() => expect(api.followupThread).toHaveBeenCalledWith("root1", { question: "예를 들어 줘", mode: "question" }));
    expect(await within(dialog).findByText("reply answer 11")).toBeInTheDocument();
    expect(within(dialog).getByText("예를 들어 줘")).toBeInTheDocument();

    fireEvent.click(within(dialog).getByRole("button", { name: "더 쉽게" }));
    await waitFor(() => expect(api.followupThread).toHaveBeenLastCalledWith("root1", { mode: "easier" }));
    expect(await within(dialog).findByText("reply answer 12")).toBeInTheDocument();
  });
});

describe("failed thread roots (502 {question})", () => {
  beforeEach(() => {
    questions = [];
    seq = 20;
  });

  it("opens the failed root and retries it as a follow-up instead of starting a new root", async () => {
    const failed = q({ id: "rootF", quote: selection.text, mode: "easier", status: "failed", answer: "", error: "" });
    api.startThread.mockImplementationOnce(async () => {
      questions = [...questions, failed];
      throw new PunisherApiError(502, "HTTP_ERROR", "HTTP 502", undefined, [], { question: failed });
    });
    renderHarness();
    fireEvent.click(within(toolbar()).getByRole("button", { name: "쉽게" }));
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("답을 만들지 못했어요")).toBeInTheDocument();

    fireEvent.click(within(dialog).getByRole("button", { name: "다시 시도" }));
    await waitFor(() => expect(api.followupThread).toHaveBeenCalledWith("rootF", { mode: "easier" }));
    expect(api.startThread).toHaveBeenCalledTimes(1);
    expect(await within(dialog).findByText("reply answer 21")).toBeInTheDocument();
  });

  it("counts only live passage roots; orphan follow-ups stay out of passages", () => {
    const list = [
      q({ id: "f1", quote: "구절", status: "failed" }),
      q({ id: "f2", quote: "구절", status: "failed", createdAt: "2026-11-20T00:02:00Z" }),
      q({ id: "ok", quote: "구절", createdAt: "2026-11-20T00:03:00Z" }),
      q({ id: "rescued", quote: "다른 구절", status: "failed" }),
      q({ id: "r1", parentId: "rescued", createdAt: "2026-11-20T00:04:00Z" }),
      q({ id: "orphan", parentId: "gone" }),
      q({ id: "legacy", question: "예전 질문" }),
    ];
    const passages = groupPassages(groupThreads(list));
    expect(countPassageThreads(passages)).toBe(2);
    expect(passages.find((p) => p.quote === "구절")?.exchanges).toBe(1);
    const ids = passageQuestionIds(passages);
    expect(ids.has("orphan")).toBe(false);
    expect(ids.has("legacy")).toBe(false);
    expect(ids.has("r1")).toBe(true);
  });
});

describe("per-thread state", () => {
  beforeEach(() => {
    seq = 30;
    questions = [
      q({ id: "root1", quote: "TCP 는 순서를 보장한다", mode: "easier", answer: "첫 답" }),
      q({ id: "root2", quote: "UDP 는 보장하지 않는다", mode: "easier", answer: "둘째 스레드 답", createdAt: "2026-11-20T00:05:00Z" }),
    ];
  });

  it("resets the follow-up input when another thread opens", async () => {
    renderHarness("root1", ["root2"]);
    fireEvent.click(await screen.findByRole("button", { name: "open-thread" }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.change(within(dialog).getByRole("textbox", { name: "이어서 묻기" }), { target: { value: "쓰다 만 질문" } });
    fireEvent.click(screen.getByRole("button", { name: "open-root2", hidden: true }));
    expect(await within(dialog).findByText("둘째 스레드 답")).toBeInTheDocument();
    expect(within(dialog).getByRole("textbox", { name: "이어서 묻기" })).toHaveValue("");
  });

  it("shows the deep-lecture status only on the thread that started it", async () => {
    renderHarness("root1", ["root2"]);
    fireEvent.click(await screen.findByRole("button", { name: "open-thread" }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "이 대화로 심화 강의 만들기" }));
    await waitFor(() => expect(api.startDeepLecture).toHaveBeenCalledWith("root1"));
    expect(await within(dialog).findByText(/심화 강의를 만들고 있어요/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "open-root2", hidden: true }));
    expect(await within(dialog).findByText("둘째 스레드 답")).toBeInTheDocument();
    expect(within(dialog).queryByText(/심화 강의를 만들고 있어요/)).toBeNull();
    expect(within(dialog).getByRole("button", { name: "이 대화로 심화 강의 만들기" })).toBeEnabled();
  });
});
