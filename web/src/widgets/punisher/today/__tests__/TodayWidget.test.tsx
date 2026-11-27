import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, configure, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ActivityResponse, LearningSettingsResponse, Recommendation, ReviewSummary, StudyRequest, TodayResponse } from "@/src/shared/api/punisher";

const rec = (patch: Partial<Recommendation>): Recommendation => ({
  kind: "goal_path", conceptId: "x", conceptName: "x", area: "network", depth: 1, title: "", reason: "", practicalSituation: "", estimatedMinutes: 10, ...patch,
});

let today: TodayResponse;
let summary: ReviewSummary;
let activity: ActivityResponse;
let requests: StudyRequest[];

const api = {
  getToday: vi.fn(async () => today),
  getReviewSummary: vi.fn(async () => summary),
  getLearningSettings: vi.fn(async (): Promise<LearningSettingsResponse> => ({ settings: { dailyMinutes: 30, readerLevel: "practitioner" }, exampleLanguages: [] }) as unknown as LearningSettingsResponse),
  listStudyRequests: vi.fn(async () => requests),
  listInquiries: vi.fn(async () => []),
  listConcepts: vi.fn(async () => []),
  dismissConcept: vi.fn(async () => undefined),
  restoreConcept: vi.fn(async () => undefined),
  dismissLecture: vi.fn(async () => undefined),
  restoreLecture: vi.fn(async () => undefined),
};
const study = { activity: vi.fn(async () => activity) };

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  const proxy = <T extends object>(target: T, fakes: Record<string, unknown>) =>
    new Proxy(target, { get: (t, key) => (typeof key === "string" && key in fakes ? fakes[key] : Reflect.get(t, key)) });
  return { ...actual, punisherApi: proxy(actual.punisherApi, api), studyApi: proxy(actual.studyApi, study) };
});

const { TodayWidget } = await import("../TodayWidget");

// 첫 렌더는 부품 모듈을 불러오느라 느리다 — 상자 세 개의 쿼리가 다 끝날 때까지 기다린다.
configure({ asyncUtilTimeout: 4000 });

function renderToday() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <TodayWidget />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const box = (name: string) => screen.findByRole("region", { name });
const group = (name: string) => screen.findByRole("group", { name });

async function openMenu(label: string) {
  const trigger = await screen.findByRole("button", { name: label });
  act(() => {
    fireEvent.pointerDown(trigger, { button: 0, pointerType: "mouse" });
    fireEvent.click(trigger);
  });
  return screen.findByRole("menu", { name: label });
}

beforeEach(() => {
  today = {
    items: [
      rec({ kind: "resume", conceptId: "net.cache", conceptName: "HTTP 캐시", lectureId: "l12", title: "Cache-Control 로 응답 재사용하기", percent: 40, anchor: "p7" }),
      rec({ conceptId: "net.etag", conceptName: "ETag 와 조건부 요청", lectureId: "l31", reason: "읽던 강의 다음", estimatedMinutes: 12 }),
      rec({ kind: "new_goal", conceptId: "db.index", conceptName: "SQL 인덱스 기초", reason: "새 목표", estimatedMinutes: 15 }),
    ],
    exhausted: false,
    review: { due: 12, overdue: 0, queued: 0, estimatedMinutes: 15, backlogged: false, lectureMinutes: 30, verify: 0 },
    unchecked: [{ lectureId: "l7", conceptId: "os.proc", conceptName: "프로세스", area: "os", title: "" }],
    synthesis: { due: true },
    session: { lectureMinutes: 27, checkMinutes: 5, reviewMinutes: 10 },
    nextChapter: { remainingMinutes: 95, thresholdMinutes: 420, dailyMinutes: 30 },
  };
  summary = { due: 12, overdue: 0, queued: 0, estimatedMinutes: 15, backlogged: false, throughput: 20, grader: "self", reviewMinutes: 10, verify: 0, today: null };
  activity = { days: [], today: { reviewsDone: 0, reviewsDue: 12, cardsDone: 0, cardsDue: 8, lecturesDone: 0 }, restDays: [], restToday: false, dailyMinutes: 30 };
  requests = [];
});

describe("오늘 — 상자·묶음", () => {
  it("처리할 것: 복습 묶음 숫자 줄 · 약 n분 · [복습 시작] · [카드로 복습]", async () => {
    renderToday();
    const process = await box("처리할 것");
    await waitFor(() => expect(within(process).getByText("복습 12")).toBeInTheDocument());
    expect(within(process).getByRole("heading", { name: "처리할 것" }).parentElement).toHaveTextContent("처리할 것21");
    expect(within(process).getByText(/확인 대기 1/)).toBeInTheDocument();
    expect(within(process).getByText(/카드 8/)).toBeInTheDocument();
    expect(within(process).getByText(/종합 1/)).toBeInTheDocument();
    expect(within(process).getByText("약 15분")).toBeInTheDocument();
    expect(within(process).getByRole("button", { name: "복습 시작" })).toBeInTheDocument();
    expect(within(process).getByRole("link", { name: "카드로 복습" })).toHaveAttribute("href", "/cards");
  });

  it("공부할 것: 이어보기 행 → 읽던 자리, 추천 행 → 강의 열기(강의 없으면 /start)", async () => {
    renderToday();
    const resume = await group("이어보기");
    const resumeLink = within(resume).getByRole("link", { name: /HTTP 캐시/ });
    expect(resumeLink).toHaveAttribute("href", "/lectures/l12?at=p7");
    expect(resumeLink).toHaveTextContent("Cache-Control 로 응답 재사용하기 · 40% 읽음");

    const recs = await group("오늘의 추천");
    expect(recs).toHaveTextContent("2 · 강의 약 27분 · 하루 30분 목표");
    expect(within(recs).getByRole("link", { name: /ETag 와 조건부 요청/ })).toHaveAttribute("href", "/lectures/l31");
    expect(within(recs).getByRole("link", { name: /SQL 인덱스 기초/ })).toHaveAttribute("href", "/start?concept=db.index");
    expect(within(recs).getByText("읽던 강의 다음 · 약 12분")).toBeInTheDocument();
    expect(recs).toHaveTextContent("남은 강의 약 3일 치");
    expect(within(recs).getByRole("link", { name: "새 챕터 열기" })).toHaveAttribute("href", "/chapters/new");
  });

  it("추천이 없으면 빈 상태, 소진이면 다른 문구 — 둘 다 [새 챕터 열기]", async () => {
    today = { ...today, items: [], nextChapter: undefined };
    const first = renderToday();
    expect(await screen.findByText("오늘 추천할 강의가 없어요")).toBeInTheDocument();
    first.unmount();
    today = { ...today, exhausted: true };
    renderToday();
    expect(await screen.findByText("지금 지도의 강의를 다 마쳤어요")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "새 챕터 열기" })).toHaveAttribute("href", "/chapters/new");
  });

  it("읽던 강의가 없으면 이어보기 묶음이 없다", async () => {
    today = { ...today, items: today.items.filter((item) => item.kind !== "resume") };
    renderToday();
    await group("오늘의 추천");
    expect(screen.queryByRole("group", { name: "이어보기" })).toBeNull();
  });

  it("묻기·새 주제: 전환 2 + 보내기, 공부할 주제(RSS 글에서 · 준비됨 [강의 시작])", async () => {
    requests = [
      { id: "s1", source: "rss", sourceRef: "", query: "TLS 핸드셰이크", context: null, resolvedConceptIds: ["net.tls"], status: "resolved", idempotencyKey: "", resolveAttempts: 1, learned: false, createdAt: "", updatedAt: "" },
    ];
    renderToday();
    const ask = await box("묻기·새 주제");
    expect(within(ask).getByRole("radio", { name: "질문하기" })).toBeInTheDocument();
    expect(within(ask).getByRole("radio", { name: "주제 공부하기" })).toBeInTheDocument();
    expect(within(ask).getByRole("button", { name: "보내기" })).toBeDisabled();
    const topics = await within(ask).findByRole("group", { name: "공부할 주제" });
    expect(topics).toHaveTextContent("RSS 글에서");
    expect(within(topics).getByRole("button", { name: "강의 시작" })).toBeInTheDocument();
  });
});

describe("오늘 — 복습 묶음 상태", () => {
  it("다 함", async () => {
    summary = { ...summary, today: { id: "r1", status: "finished", grader: "self", total: 15, done: 15 } };
    renderToday();
    expect(await screen.findByText("오늘 복습을 다 했어요")).toBeInTheDocument();
    expect(screen.getByText("· 15/15")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "오늘 더 하기" })).toBeInTheDocument();
  });

  it("쉬는 날", async () => {
    activity = { ...activity, restToday: true };
    renderToday();
    expect(await screen.findByText("오늘은 쉬는 날이에요")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "그래도 복습" })).toBeInTheDocument();
  });

  it("없음 — 행동 없음", async () => {
    today = { ...today, review: { ...today.review!, due: 0 }, unchecked: [], synthesis: { due: false } };
    activity = { ...activity, today: { ...activity.today, cardsDue: 0 } };
    renderToday();
    const process = await box("처리할 것");
    expect(await within(process).findByText("지금 볼 복습은 없어요")).toBeInTheDocument();
    expect(within(process).queryByRole("button")).toBeNull();
  });

  it("중간에 나감", async () => {
    summary = { ...summary, today: { id: "r1", status: "active", grader: "self", total: 15, done: 6 } };
    renderToday();
    expect(await screen.findByText("· 9개 남음")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "이어서 복습" })).toBeInTheDocument();
  });

  it("불러오기 실패면 상자 자리에 오류(숨기지 않음)", async () => {
    // 4xx 는 다시 시도하지 않는다(shouldRetry) — 바로 실패 상태.
    const { PunisherApiError } = await import("@/src/shared/api/punisher");
    api.getToday.mockRejectedValueOnce(new PunisherApiError(400, "BAD", "boom"));
    renderToday();
    const process = await box("처리할 것");
    expect(await within(process).findByRole("alert")).toBeInTheDocument();
    expect(within(await box("공부할 것")).getByRole("alert")).toBeInTheDocument();
  });
});

describe("오늘 — 토스트 되돌리기", () => {
  it("추천 ⋯ 필요 없음 → '추천에서 뺐어요 [되돌리기]'", async () => {
    renderToday();
    const menu = await openMenu("ETag 와 조건부 요청 동작");
    expect(within(menu).getByRole("menuitem", { name: "지도에서 보기" })).toHaveAttribute("href", "/map?node=net.etag");
    fireEvent.click(within(menu).getByRole("menuitem", { name: "필요 없음" }));
    await waitFor(() => expect(api.dismissConcept).toHaveBeenCalledWith("net.etag"));
    expect(await screen.findByText("추천에서 뺐어요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "되돌리기" }));
    await waitFor(() => expect(api.restoreConcept).toHaveBeenCalledWith("net.etag"));
  });

  it("이어보기 ⋯ 빼기 → '이어보기에서 뺐어요 [되돌리기]'", async () => {
    renderToday();
    const menu = await openMenu("HTTP 캐시 동작");
    fireEvent.click(within(menu).getByRole("menuitem", { name: "이어보기에서 빼기" }));
    await waitFor(() => expect(api.dismissLecture).toHaveBeenCalledWith("l12"));
    expect(await screen.findByText("이어보기에서 뺐어요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "되돌리기" }));
    await waitFor(() => expect(api.restoreLecture).toHaveBeenCalledWith("l12"));
  });
});
