import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CheckHistoryItem, Lecture, Recommendation, SectionFix } from "@/src/shared/api/punisher";
import { useScreenChrome } from "@/src/shared/lib/screen-chrome";

// ── 바꿔 가며 쓰는 상태 ──
let search = "";
let desktop = false;
let completedAt: string | undefined;
let loadError: Error | null = null;
let conceptState = "in_progress";
let todayItems: Recommendation[] = [];
let historySessions: CheckHistoryItem[] = [];
let fixes: SectionFix[] = [];
let promptOverridden = false;
const router = { push: vi.fn(), replace: vi.fn(), back: vi.fn() };
const markKnown = vi.fn(async () => ({ conceptId: "c1", state: "known" }));

function lecture(): Lecture {
  return {
    id: "l1", conceptId: "c1", objectiveIds: ["c1#1", "c1#2"], depth: 1, kind: "library", status: "published", version: 1,
    title: "Cache-Control 로 응답 재사용하기", sources: null, flagged: false, estimatedMinutes: 12, createdAt: "2026-11-01T00:00:00Z",
    progress: { anchor: "", percent: 40, startedAt: "", lastSeenAt: "", completedAt },
    content: {
      title: "Cache-Control 로 응답 재사용하기", estimatedMinutes: 12, objectiveIds: ["c1#1", "c1#2"], newScope: [], prerequisitesReferenced: [],
      sections: [
        { anchor: "s1", kind: "concept", title: "왜 다시 받지 않아도 되나요", blocks: [] },
        { anchor: "s2", kind: "concept", title: "Cache-Control 읽는 법", blocks: [] },
      ],
    } as unknown as Lecture["content"],
  };
}

function rec(partial: Partial<Recommendation>): Recommendation {
  return { kind: "goal_path", conceptId: "c9", conceptName: "ETag", area: "web", depth: 1, title: "ETag 강의", reason: "", practicalSituation: "", estimatedMinutes: 10, ...partial };
}

const idle = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(async () => undefined), reset: vi.fn(), isPending: false, isSuccess: false, error: null, data: undefined });

vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => router,
  usePathname: () => "/lectures/l1",
  useSearchParams: () => new URLSearchParams(search),
  Link: ({ href, children, className }: { href: string; children?: ReactNode; className?: string }) => (
    <a href={href} className={className}>{children}</a>
  ),
}));

vi.mock("@forgen-team/ui", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@forgen-team/ui")>()),
  useMediaQuery: (query: string) => (query === "(min-width: 1024px)" ? desktop : false),
}));

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  const api = { markKnown };
  return { ...actual, punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }) };
});

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useLecture: () => (loadError
    ? { data: undefined, isLoading: false, error: loadError, refetch: vi.fn() }
    : { data: lecture(), isLoading: false, error: null, refetch: vi.fn() }),
  useConceptDetail: () => ({ data: { id: "c1", name: "HTTP 캐시", area: "data-structures", userState: conceptState } }),
  useMastery: () => ({ data: { conceptId: "c1", objectives: [
    { objectiveId: "c1#1", nextReviewAt: "2026-12-02T12:00:00Z" },
    { objectiveId: "c1#2", nextReviewAt: "2026-12-09T12:00:00Z" },
  ] } }),
  useLectureLinks: () => ({ data: { parents: [], children: [] } }),
  usePrompts: () => ({ data: { blocks: promptOverridden ? [{ overridden: true, stage: "lecture" }] : [] } }),
  useToday: () => ({ data: { items: todayItems, exhausted: false } }),
  useCheckHistory: () => ({ data: { pages: [{ sessions: historySessions, nextCursor: "" }] } }),
  useLectureExercises: () => ({ data: { fixes } }),
  useCompleteLecture: () => idle(),
  useUncompleteLecture: () => idle(),
  useUndoKnown: () => idle(),
  useQuestions: () => ({ data: [] }),
  useNotes: () => ({ data: [{ id: "n1" }, { id: "n2" }] }),
}));

vi.mock("@/src/features/punisher/reading-assist", () => ({
  ReadingThreadsProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  useReadingThreads: () => null,
  ReadingAssist: () => null,
  PassageThreadList: () => null,
  countPassageThreads: () => 0,
  passageQuestionIds: () => new Set<string>(),
  THREAD_PANEL_QUERY: "(min-width: 1024px)",
}));
vi.mock("@/src/features/punisher/lecture-progress", () => ({
  useLectureProgress: () => ({ getAnchor: () => "s1", saveNow: vi.fn() }),
  scrollToAnchor: vi.fn(),
  anchorElementId: (anchor: string) => `pa-${anchor}`,
}));
vi.mock("@/src/features/punisher/check-start", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/features/punisher/check-start")>()),
  useLaunchCheck: () => ({ launch: vi.fn(), isPending: false, error: null }),
}));
vi.mock("@/src/features/punisher/pretest", () => ({ PretestPanel: () => null, usePretest: () => ({ data: undefined }) }));
vi.mock("@/src/features/punisher/level-suggestion", () => ({ LevelSuggestionCard: () => null }));
vi.mock("@/src/features/punisher/lecture-exercise", () => ({ ExerciseProvider: ({ children }: { children: ReactNode }) => <>{children}</> }));
vi.mock("@/src/features/punisher/reading-signals", () => ({ ReadingSignalsProvider: ({ children }: { children: ReactNode }) => <>{children}</> }));
vi.mock("../ui/LectureBody", () => ({ LectureBody: () => <p>본문</p> }));
vi.mock("../ui/DeepLectureStatus", () => ({ DeepLectureStatus: () => null }));
vi.mock("../ui/QuestionsPanel", () => ({ QuestionsPanel: () => <p>질문 목록</p> }));
vi.mock("../ui/NotesPanel", () => ({ NotesPanel: () => <p>메모 목록</p> }));
vi.mock("../ui/FixesSheet", () => ({ FixesSheet: ({ open }: { open: boolean }) => (open ? <p>고칠 곳 시트</p> : null) }));
vi.mock("../ui/RegenerateSheet", () => ({ RegenerateSheet: ({ open }: { open: boolean }) => (open ? <p>다시 만들기 시트</p> : null) }));

const { LectureViewer } = await import("../LectureViewer");

/** 틀이 그릴 페이지 동작을 읽어 둔다. */
let chrome: ReturnType<typeof useScreenChrome> = { title: null, subtitle: null, actions: null };
function ChromeProbe() {
  chrome = useScreenChrome();
  return null;
}

function renderViewer() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ChromeProbe />
      <LectureViewer lectureId="l1" />
    </QueryClientProvider>,
  );
}

function wrapup() {
  return screen.getByRole("region", { name: "마무리" });
}

describe("강의 화면(재설계 안 B)", () => {
  beforeEach(() => {
    search = "";
    desktop = false;
    completedAt = undefined;
    loadError = null;
    conceptState = "in_progress";
    todayItems = [rec({ kind: "resume", conceptId: "c5", conceptName: "이어보기", lectureId: "l5" }), rec({ lectureId: "l9" })];
    historySessions = [];
    fixes = [];
    promptOverridden = false;
  });

  it("화면 제목만 넘기고 자체 h1·← ·'학습 상태' 메뉴는 그리지 않는다", () => {
    renderViewer();
    expect(chrome.title).toBe("Cache-Control 로 응답 재사용하기");
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
    expect(screen.queryByText("학습 상태")).toBeNull();
    expect(screen.queryByRole("link", { name: /뒤로|돌아가기/ })).toBeNull();
  });

  it("메타 한 줄: 영역 · 약 n분", () => {
    renderViewer();
    expect(screen.getByText("자료구조 · 약 12분")).toBeInTheDocument();
  });

  it("⋯ 페이지 동작: 이미 앎 · 지도에서 보기(?node=) · 강의 프롬프트 — 피드백·더 깊게·예시 언어·검색은 없다", () => {
    renderViewer();
    expect(chrome.actions?.map((a) => a.label)).toEqual(["이미 앎", "지도에서 보기", "강의 프롬프트"]);
    expect(chrome.actions?.find((a) => a.id === "map")?.href).toBe("/map?node=c1");
    expect(chrome.actions?.every((a) => a.kind === undefined)).toBe(true);
  });

  it("⋯ 조건 항목: 고칠 곳 n · 새 프롬프트로 다시 만들기를 순서대로 넣는다", () => {
    fixes = [{ section: "s1", reasons: [] }, { section: "s2", reasons: [] }] as unknown as SectionFix[];
    promptOverridden = true;
    renderViewer();
    expect(chrome.actions?.map((a) => a.label)).toEqual(["이미 앎", "고칠 곳 2", "새 프롬프트로 다시 만들기", "지도에서 보기", "강의 프롬프트"]);
    act(() => chrome.actions?.find((a) => a.id === "fixes")?.onSelect?.());
    expect(screen.getByText("고칠 곳 시트")).toBeInTheDocument();
  });

  it("⋯ 이미 아는 개념이면 '이미 앎'을 넣지 않는다", () => {
    conceptState = "known";
    renderViewer();
    expect(chrome.actions?.map((a) => a.label)).toEqual(["지도에서 보기", "강의 프롬프트"]);
  });

  it("이미 앎: 누르면 바로 표시(짧은 확인은 대기열로) + 되돌리기 토스트", async () => {
    renderViewer();
    act(() => chrome.actions?.find((a) => a.id === "known")?.onSelect?.());
    await waitFor(() => expect(markKnown).toHaveBeenCalledWith("c1", { queueVerify: true }));
    expect(await screen.findByText("이미 앎으로 표시했어요")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "되돌리기" })).toBeInTheDocument();
    expect(screen.queryByText(/짧게 확인할까요/)).toBeNull();
  });

  it("끝 마무리 — 읽기 전: [완료로 표시] [확인하기], 이미 앎·다음 강의 없음", () => {
    renderViewer();
    const box = within(wrapup());
    expect(box.getAllByRole("button").map((b) => b.textContent)).toEqual(["완료로 표시", "확인하기"]);
    expect(box.queryByText("이미 앎")).toBeNull();
    expect(box.queryByRole("link")).toBeNull();
  });

  it("끝 마무리 — 완료 뒤: 완료·목표 수 + [확인하기] + [다음 강의: 첫 추천](이어보기 제외)", () => {
    completedAt = "2026-11-20T00:00:00Z";
    renderViewer();
    const box = within(wrapup());
    expect(box.getByText("완료했어요 · 학습 목표 2개")).toBeInTheDocument();
    expect(box.getByRole("button", { name: "확인하기" })).toBeInTheDocument();
    expect(box.getByRole("link", { name: "다음 강의: ETag" }).getAttribute("href")).toBe("/lectures/l9");
    expect(box.queryByRole("button", { name: "완료로 표시" })).toBeNull();
  });

  it("끝 마무리 — 다음 추천에 강의가 없으면 /start?concept=, 추천이 없으면 [오늘로]", () => {
    completedAt = "2026-11-20T00:00:00Z";
    todayItems = [rec({ lectureId: undefined, conceptId: "c9" })];
    const { unmount } = renderViewer();
    expect(within(wrapup()).getByRole("link", { name: "다음 강의: ETag" }).getAttribute("href")).toBe("/start?concept=c9");
    unmount();
    todayItems = [rec({ lectureId: "l1", conceptId: "c1" })];
    renderViewer();
    expect(within(wrapup()).getByRole("link", { name: "오늘로" }).getAttribute("href")).toBe("/");
  });

  it("끝 마무리 — 확인 뒤: 결과 · 다음 복습 날짜 + [다음 강의]만", () => {
    completedAt = "2026-11-20T00:00:00Z";
    historySessions = [{ id: "s1", kind: "check", grader: "llm", status: "finished", lectureId: "l1", itemCount: 5, doneCount: 5, unaidedCorrect: 4, createdAt: "" }];
    renderViewer();
    const box = within(wrapup());
    expect(box.getByText("확인 결과 4/5 · 다음 복습 12월 2일")).toBeInTheDocument();
    expect(box.queryByRole("button", { name: "확인하기" })).toBeNull();
    expect(box.getByRole("link", { name: "다음 강의: ETag" })).toBeInTheDocument();
  });

  it("폰: 본문 고정 바는 [목차] [질문과 메모 n] 두 버튼, 누르면 ?panel= 로 push", () => {
    renderViewer();
    const bar = document.querySelector("[data-lecture-readbar]") as HTMLElement;
    expect(within(bar).getAllByRole("button").map((b) => b.textContent)).toEqual(["목차", "질문과 메모 2"]);
    fireEvent.click(within(bar).getByRole("button", { name: "목차" }));
    expect(router.push).toHaveBeenCalledWith("/lectures/l1?panel=toc", expect.anything());
  });

  it("폰: ?panel=notes 로 들어오면 질문과 메모 시트가 열리고, 닫으면 panel 만 지운다", () => {
    search = "panel=notes&at=s2";
    renderViewer();
    const sheet = screen.getByRole("dialog", { name: "질문과 메모" });
    expect(within(sheet).getByText("메모 목록")).toBeInTheDocument();
    fireEvent.click(within(sheet).getByRole("button", { name: /닫기/ }));
    expect(router.replace).toHaveBeenCalledWith("/lectures/l1?at=s2", expect.anything());
  });

  it("데스크톱: 고정 바 없이 보조 열 탭, ?panel=notes 면 그 탭", () => {
    desktop = true;
    search = "panel=notes";
    renderViewer();
    expect(document.querySelector("[data-lecture-readbar]")).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByRole("radio", { name: "질문과 메모 2" })).toBeChecked();
    expect(screen.getByText("메모 목록")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("radio", { name: "목차" }));
    expect(screen.getByRole("navigation", { name: "강의 목차" })).toBeInTheDocument();
  });

  it("실패: '강의를 불러오지 못했어요' [다시 시도], 돌아가기 링크 없음", () => {
    loadError = new Error("boom");
    renderViewer();
    expect(screen.getByText("강의를 불러오지 못했어요")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "다시 시도" })).toBeInTheDocument();
    expect(screen.queryByRole("link")).toBeNull();
  });
});
