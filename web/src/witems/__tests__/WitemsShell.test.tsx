import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { UIProvider } from "@forgen-team/ui";
import { setWitemsRuntime } from "@/src/shared/config/runtime";
import { Link } from "@/src/shared/lib/router";
import { AppRoutes } from "@/src/app/App";
import { WitemsShell } from "../WitemsShell";

vi.mock("@/src/entities/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/entities/punisher")>();
  return {
    lectureHref: actual.lectureHref,
    startLectureHref: actual.startLectureHref,
    LocaleSync: () => null,
    useActivity: () => ({ data: { restToday: false, today: { reviewsDue: 1, cardsDue: 1 } } }),
    useToday: () => ({ data: { items: [{ kind: "goal_path", conceptId: "c1", conceptName: "TCP", title: "TCP" }], exhausted: false, unchecked: [] } }),
    useStartReview: () => ({ mutate: vi.fn() }),
  };
});

vi.mock("@/src/widgets/claude-status/ClaudeStatusBanner", () => ({
  ClaudeStatusBanner: () => <p>claude-banner</p>,
  useClaudeStatusIssue: () => true,
}));

vi.mock("@/src/pages", () => {
  const page = (name: string) => function Screen() { return <p>{name}</p>; };
  return {
    TodayPage: page("today-screen"), HistoryPage: page("history-screen"), MapPage: page("map-screen"), LecturePage: page("lecture-screen"),
    InquiryPage: page("inquiry-screen"), CheckPage: page("check-screen"), InstantPage: page("instant-screen"), CardsPage: page("cards-screen"),
    StartPage: page("start-screen"), NotFoundPage: page("not-found-screen"), SettingsPage: page("settings-screen"),
    PromptSettingsPage: page("prompts-screen"), NewChapterPage: page("new-chapter-screen"),
  };
});

function mountAt(path: string) {
  const runtime = { apiBase: "/apps-api/punisher/", portal: document.body, setTitle: vi.fn(), setActions: vi.fn(), setCommands: vi.fn(), setBadges: vi.fn() };
  setWitemsRuntime(runtime);
  render(
    <UIProvider linkAs={Link} inShell>
      <MemoryRouter initialEntries={[path]}>
        <AppRoutes shell={<WitemsShell />} />
      </MemoryRouter>
    </UIProvider>,
  );
  return runtime;
}

describe("WitemsShell (inside the shell)", () => {
  afterEach(() => setWitemsRuntime(null));

  it("titles 오늘, sends punisher 설정 (no 검색), the badge and the banner at the top of the body", () => {
    const runtime = mountAt("/");
    expect(runtime.setTitle).toHaveBeenLastCalledWith("오늘");
    const actions = runtime.setActions.mock.lastCall?.[0] as Array<{ id: string; label: string; kind?: string }>;
    expect(actions.map((a) => a.label)).toEqual(["punisher 설정"]);
    expect(actions.some((a) => a.id === "search")).toBe(false);
    expect(runtime.setBadges).toHaveBeenLastCalledWith({ "/": 2 });
    // 머리는 셸이 그린다 — 앱은 h1 을 그리지 않는다
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
    expect(screen.getByText("claude-banner")).toBeInTheDocument();
    const commands = runtime.setCommands.mock.lastCall?.[0] as Array<{ id: string; href?: string }>;
    expect(commands.find((c) => c.id === "next-lecture")?.href).toBe("/start?concept=c1");
  });

  it("draws no app frame and no palette — ⌘K belongs to the shell, which gets the same command list", () => {
    const runtime = mountAt("/map");
    expect(screen.queryByRole("navigation", { name: "punisher 섹션" })).toBeNull();
    expect(screen.queryByRole("button", { name: /^명령/ })).toBeNull();
    fireEvent.keyDown(window, { key: "k", code: "KeyK", metaKey: true });
    expect(screen.queryByRole("dialog")).toBeNull();
    const commands = runtime.setCommands.mock.lastCall?.[0] as Array<{ id: string; onSelect?: () => void }>;
    expect(commands.map((c) => c.id)).toContain("history");
    expect(typeof commands.find((c) => c.id === "review")?.onSelect).toBe("function");
  });

  it("leaves the settings action out on the settings screen", () => {
    const runtime = mountAt("/settings");
    expect(runtime.setTitle).toHaveBeenLastCalledWith("punisher 설정");
    expect(runtime.setActions).toHaveBeenLastCalledWith([]);
    expect(screen.queryByText("claude-banner")).toBeNull();
  });

  it("serves /start as a common route", () => {
    mountAt("/start?concept=c1");
    expect(screen.getByText("start-screen")).toBeInTheDocument();
  });
});
