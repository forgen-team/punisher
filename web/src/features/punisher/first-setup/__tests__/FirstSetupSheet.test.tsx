import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Concept, LearningSettings, LearningSettingsPatch, LearningSettingsResponse, ReaderLevel } from "@/src/shared/api/punisher";

const baseSettings: LearningSettings = {
  exampleLanguage: "typescript",
  dailyMinutes: 30,
  masteryTarget: 4,
  reviewIntervals: [1, 3, 7],
  reviewMinutes: 10,
  reviewGrader: "self",
  pretestEnabled: true,
  labOs: "macos",
  adaptiveIntervals: false,
  locale: "ko",
  readerLevel: "",
  areaFamiliarity: {},
};

const concept = (over: Partial<Concept>): Concept => ({
  id: "x", name: "x", area: "os", depth: 1, summary: "", practicalCases: null, docLinks: null,
  status: "active", origin: "seed", objectiveCount: 1, userState: "not_started", updatedAt: "", ...over,
});

const concepts: Concept[] = [
  concept({ id: "os.process", area: "os" }),
  concept({ id: "os.thread", area: "os" }),
  concept({ id: "os.known", area: "os", userState: "known" }),
  concept({ id: "os.deep", area: "os", depth: 2 }),
  concept({ id: "os.proposed", area: "os", status: "proposed" }),
  concept({ id: "db.index", area: "database" }),
];

let current: LearningSettings = baseSettings;
const api = {
  getLearningSettings: vi.fn(async (): Promise<LearningSettingsResponse> => ({ settings: current, exampleLanguages: [] })),
  saveLearningSettings: vi.fn(async (patch: LearningSettingsPatch): Promise<LearningSettingsResponse> => {
    current = { ...current, ...patch };
    return { settings: current, exampleLanguages: [] };
  }),
  listConcepts: vi.fn(async () => concepts),
  markKnown: vi.fn<(id: string, options?: { queueVerify?: boolean }) => Promise<unknown>>(async () => ({})),
};

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return { ...actual, punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }) };
});

// BottomSheet 스크롤 잠금 해제가 window.scrollTo 를 부른다(jsdom 미구현).
Object.defineProperty(window, "scrollTo", { value: () => undefined, writable: true });

const { FirstSetup } = await import("../FirstSetupSheet");
const { closedRecently } = await import("../useFirstSetup");

function renderSetup(level: ReaderLevel = "", path = "/") {
  current = { ...baseSettings, readerLevel: level };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={client}>
        <FirstSetup />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const title = (name: string) => screen.findByRole("heading", { name: new RegExp(name) });
const button = (name: string) => screen.getByRole("button", { name });

describe("처음 설정 시트", () => {
  beforeEach(() => {
    window.sessionStorage.clear();
    window.localStorage.clear();
  });

  it("does not open once a reader level is chosen", async () => {
    renderSetup("practitioner");
    await waitFor(() => expect(api.getLearningSettings).toHaveBeenCalled());
    expect(screen.queryByRole("heading", { name: /개발은 어느 정도/ })).toBeNull();
  });

  it("moves through the three steps and derives area defaults from the level", async () => {
    renderSetup();
    await title("처음 설정 · 1 / 3");
    expect(button("다음")).toBeDisabled();

    fireEvent.click(screen.getByRole("radio", { name: /CS 기본기는 탄탄해요/ }));
    fireEvent.click(button("다음"));
    await title("영역마다 아는 만큼");
    const osRow = screen.getByRole("radiogroup", { name: "운영체제" });
    expect(within(osRow).getByRole("radio", { name: "설명 가능" })).toBeChecked();

    fireEvent.click(button("이전"));
    await title("처음 설정 · 1 / 3");
    expect(screen.getByRole("radio", { name: /CS 기본기는 탄탄해요/ })).toBeChecked();

    fireEvent.click(screen.getByRole("radio", { name: /개발은 처음이에요/ }));
    fireEvent.click(button("다음"));
    expect(within(screen.getByRole("radiogroup", { name: "운영체제" })).getByRole("radio", { name: "처음" })).toBeChecked();

    fireEvent.click(button("다음"));
    await title("이렇게 시작할게요");
    expect(screen.getByText("여기서 시작")).toBeInTheDocument();
    expect(screen.getByText("없음")).toBeInTheDocument();
  });

  it("skip saves practitioner, closes, and asks again after two weeks", async () => {
    renderSetup();
    await title("처음 설정");
    fireEvent.click(button("건너뛰기"));
    await waitFor(() => expect(api.saveLearningSettings).toHaveBeenCalledWith({ readerLevel: "practitioner" }));
    await waitFor(() => expect(screen.queryByRole("heading", { name: /처음 설정/ })).toBeNull());
    expect(api.markKnown).not.toHaveBeenCalled();
    expect(window.localStorage.getItem("punisher.firstSetup.closed")).not.toBeNull();
  });

  it("treats × the same as skip", async () => {
    renderSetup();
    await title("처음 설정");
    fireEvent.keyDown(document.activeElement ?? document.body, { key: "Escape" });
    await waitFor(() => expect(api.saveLearningSettings).toHaveBeenCalledWith({ readerLevel: "practitioner" }));
    await waitFor(() => expect(screen.queryByRole("heading", { name: /처음 설정/ })).toBeNull());
    expect(window.localStorage.getItem("punisher.firstSetup.closed")).not.toBeNull();
  });

  it("re-opens for a skipped user once two weeks have passed", async () => {
    window.localStorage.setItem("punisher.firstSetup.closed", String(Date.now() - 15 * 24 * 60 * 60 * 1000));
    renderSetup("practitioner");
    await title("처음 설정 · 1 / 3");
  });

  it("opens from ?first=1 even when a level is chosen", async () => {
    renderSetup("practitioner", "/?first=1");
    await title("처음 설정 · 1 / 3");
  });

  it("finish saves settings, then marks depth-1 concepts of 'can explain' areas known", async () => {
    renderSetup();
    await title("처음 설정");
    fireEvent.click(screen.getByRole("radio", { name: /웹 개발을 실무로 해요/ }));
    fireEvent.click(button("다음"));
    await title("영역마다 아는 만큼");
    fireEvent.click(within(screen.getByRole("radiogroup", { name: "운영체제" })).getByRole("radio", { name: "설명 가능" }));
    fireEvent.click(button("다음"));
    await title("이렇게 시작할게요");
    expect(screen.getByText("운영체제 — 깊이 1 이미 앎")).toBeInTheDocument();
    expect(screen.getByText("숨김")).toBeInTheDocument();

    fireEvent.click(button("시작하기"));
    await waitFor(() => expect(api.markKnown).toHaveBeenCalledTimes(2));
    const patch = api.saveLearningSettings.mock.calls[0][0];
    expect(patch.readerLevel).toBe("practitioner");
    expect(patch.areaFamiliarity).toMatchObject({ os: "can_explain", database: "heard", "network-web": "heard" });
    expect(api.markKnown.mock.calls.map(([id]) => id).sort()).toEqual(["os.process", "os.thread"]);
    expect(api.saveLearningSettings.mock.invocationCallOrder[0]).toBeLessThan(api.markKnown.mock.invocationCallOrder[0]);
    await waitFor(() => expect(screen.queryByRole("heading", { name: /이렇게 시작할게요/ })).toBeNull());
  });

  it("keeps the sheet open with a retry when some concepts fail to save", async () => {
    api.markKnown.mockImplementation(async (id) => {
      if (id === "os.thread") { throw new Error("boom"); }
      return {};
    });
    renderSetup();
    await title("처음 설정");
    fireEvent.click(screen.getByRole("radio", { name: /CS 기본기는 탄탄해요/ }));
    fireEvent.click(button("다음"));
    await title("영역마다 아는 만큼");
    fireEvent.click(button("다음"));
    await title("이렇게 시작할게요");
    fireEvent.click(button("시작하기"));
    expect(await screen.findByRole("alert")).toHaveTextContent("1개를 이미 앎으로 저장하지 못했어요");

    api.markKnown.mockImplementation(async () => ({}));
    fireEvent.click(button("다시 시도"));
    await waitFor(() => expect(api.markKnown).toHaveBeenLastCalledWith("os.thread", { queueVerify: true }));
    await waitFor(() => expect(screen.queryByRole("heading", { name: /이렇게 시작할게요/ })).toBeNull());
  });
});

describe("closedRecently", () => {
  const day = 24 * 60 * 60 * 1000;
  it("keeps the sheet closed for two weeks after X", () => {
    const now = 100 * day;
    expect(closedRecently(null, now)).toBe(false);
    expect(closedRecently(String(now - 13 * day), now)).toBe(true);
    expect(closedRecently(String(now - 15 * day), now)).toBe(false);
    expect(closedRecently("1", now)).toBe(true);
    expect(closedRecently("x", now)).toBe(false);
  });
});
