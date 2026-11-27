import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { LearningSettings, StudyPrefs } from "@/src/shared/api/punisher";
import type { AppStatus } from "@/src/shared/api/status";

const flags = vi.hoisted(() => ({ witems: false }));
const saveSettings = vi.hoisted(() => vi.fn());
const savePrefs = vi.hoisted(() => vi.fn());

const settings: LearningSettings = {
  exampleLanguage: "typescript",
  dailyMinutes: 30,
  masteryTarget: 3,
  reviewIntervals: [1, 3, 7],
  reviewMinutes: 10,
  reviewGrader: "llm",
  pretestEnabled: true,
  labOs: "macos",
  adaptiveIntervals: true,
  locale: "ko",
  readerLevel: "",
  areaFamiliarity: {},
};
const prefs: StudyPrefs = { memoryModel: "fsrs", targetRetention: 0.9, restDays: [0, 6], reminderHour: -1 };
const status: AppStatus = {
  claude: { ready: true, installed: true, version: "2.1.0", loggedIn: true, active: 1, queued: 0 },
  provider: "claude",
  providers: [{ id: "claude", ready: true, installed: true, version: "2.1.0", loggedIn: true }],
  appVersion: "1.2.3",
  dataDir: "/tmp/punisher",
};

vi.mock("@/src/shared/config/runtime", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/config/runtime")>()),
  isWitems: () => flags.witems,
}));

vi.mock("@/src/shared/api/status", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/api/status")>()),
  useAppStatus: () => ({ data: status, isLoading: false, isFetching: false, refetch: vi.fn() }),
}));

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useLearningSettings: () => ({
    isLoading: false,
    error: null,
    refetch: vi.fn(),
    data: { settings, exampleLanguages: [{ id: "typescript", label: "TypeScript" }, { id: "go", label: "Go" }] },
  }),
  useSaveLearningSettings: () => ({ mutateAsync: saveSettings, mutate: vi.fn(), isPending: false, variables: undefined, error: null }),
  useStudyPrefs: () => ({ isLoading: false, error: null, refetch: vi.fn(), data: prefs }),
  useSaveStudyPrefs: () => ({ mutateAsync: savePrefs }),
}));

const { SettingsPage } = await import("../SettingsPage");

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <SettingsPage />
    </QueryClientProvider>,
  );
}

const headings = () => screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);

describe("설정 화면", () => {
  beforeEach(() => {
    flags.witems = false;
    saveSettings.mockResolvedValue(settings);
    savePrefs.mockResolvedValue(prefs);
  });

  it("단독판은 7구역을 학습 · 복습 · 읽기 · AI 연결 · 고급 · 화면 · 정보 순서로 그린다(자체 h1 없음)", () => {
    renderPage();
    expect(headings()).toEqual(["학습", "복습", "읽기", "AI 연결", "고급", "화면", "정보"]);
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
  });

  it("셸 안은 화면(테마) 구역 없이 6구역이고, 정보에 데이터 위치를 두지 않는다", () => {
    flags.witems = true;
    renderPage();
    expect(headings()).toEqual(["학습", "복습", "읽기", "AI 연결", "고급", "정보"]);
    expect(screen.queryByText("/tmp/punisher")).toBeNull();
    expect(screen.getByText("1.2.3")).toBeInTheDocument();
  });

  it("복습 구역 한 곳에 9항목이 시안 순서로 모여 있다(학습 구역에는 복습 항목이 없다)", () => {
    renderPage();
    const review = screen.getByRole("heading", { name: "복습" }).closest("section");
    const learning = screen.getByRole("heading", { name: "학습" }).closest("section");
    expect(review).not.toBeNull();
    expect(learning).not.toBeNull();
    const inReview = within(review as HTMLElement);
    // 행 이름(눈에 보이는 글자)이 위에서부터 시안 순서 9개.
    const rowNames = Array.from((review as HTMLElement).querySelectorAll('span[aria-hidden="true"]')).map((el) => el.textContent);
    expect(rowNames).toEqual([
      "숙달 기준", "복습 간격 방식", "복습 간격(일)", "목표 유지율", "하루 복습 시간",
      "복습 간격 자동 조절", "복습 채점 방식", "쉬는 요일", "복습 알림",
    ]);
    // 각 행의 조작도 그 이름으로 찾아진다(스크린리더).
    for (const name of ["숙달 기준", "복습 간격 방식", "목표 유지율", "하루 복습 시간", "복습 채점 방식", "복습 알림"]) {
      expect(inReview.getByRole("combobox", { name })).toBeInTheDocument();
    }
    expect(inReview.getByRole("textbox", { name: "복습 간격(일)" })).toBeInTheDocument();
    expect(inReview.getByRole("switch", { name: "복습 간격 자동 조절" })).toBeInTheDocument();
    expect(inReview.getByRole("group", { name: "쉬는 요일" })).toBeInTheDocument();
    const inLearning = within(learning as HTMLElement);
    for (const name of [/복습 간격/, /숙달 기준/, /복습 채점/, /하루 복습 시간/, /쉬는 요일/]) {
      expect(inLearning.queryByText(name)).toBeNull();
    }
    // 바로 저장 구역에는 저장 버튼이 없다.
    expect(inReview.queryByRole("button", { name: /^저장$/ })).toBeNull();
  });

  it("스위치를 켜고 끄면 그 칸만 바로 저장하고 그 자리에 저장했어요를 잠깐 보인다", async () => {
    renderPage();
    const toggle = screen.getByRole("switch", { name: "복습 간격 자동 조절" });
    fireEvent.click(toggle);
    expect(saveSettings).toHaveBeenCalledWith({ adaptiveIntervals: false });
    expect(await screen.findByText("저장했어요")).toBeInTheDocument();
  });

  it("쉬는 요일 칩은 study-prefs 로 저장하고, 복습 간격(일) 칸은 칸을 떠날 때 저장한다", async () => {
    renderPage();
    fireEvent.click(screen.getByRole("group", { name: "쉬는 요일" }).querySelector("button, [role=checkbox], label") as HTMLElement);
    await waitFor(() => expect(savePrefs).toHaveBeenCalled());

    const intervals = screen.getByRole("textbox", { name: "복습 간격(일)" });
    fireEvent.change(intervals, { target: { value: "1, 2, 5" } });
    expect(saveSettings).not.toHaveBeenCalledWith({ reviewIntervals: [1, 2, 5] });
    fireEvent.blur(intervals);
    await waitFor(() => expect(saveSettings).toHaveBeenCalledWith({ reviewIntervals: [1, 2, 5] }));
  });
});
