import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation } from "react-router-dom";
import { UIProvider } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { AppRoutes } from "../App";
import { AppShell } from "../AppShell";

const state = vi.hoisted(() => ({ issue: false, select: vi.fn() }));

vi.mock("@/src/entities/punisher", () => ({
  LocaleSync: () => null,
  useActivity: () => ({ data: { restToday: false, today: { reviewsDue: 2, cardsDue: 0 } } }),
  useToday: () => ({ data: { items: [], exhausted: false, unchecked: [{ lectureId: "l1" }] } }),
}));

vi.mock("@/src/widgets/claude-status/ClaudeStatusBanner", () => ({
  ClaudeStatusBanner: () => <p>claude-banner</p>,
  useClaudeStatusIssue: () => state.issue,
}));

vi.mock("@/src/pages", async () => {
  const { usePageActions } = await import("@/src/shared/lib/screen-chrome");
  const { Link: AppLink } = await import("@/src/shared/lib/router");
  const page = (name: string) => function Screen() { return <p>{name}</p>; };
  return {
    TodayPage: page("today-screen"),
    HistoryPage: () => <AppLink href="/chapters/new">to-new-chapter</AppLink>,
    MapPage: function MapScreen() {
      usePageActions([
        { id: "new-chapter", label: "새 챕터", icon: "plus", kind: "default", href: "/chapters/new" },
        { id: "goals", label: "목표 관리", href: "/map?panel=goals" },
        { id: "drop", label: "지우기", danger: true, onSelect: state.select },
      ]);
      return <p>map-screen</p>;
    },
    LecturePage: page("lecture-screen"),
    InquiryPage: page("inquiry-screen"),
    CheckPage: page("check-screen"),
    InstantPage: page("instant-screen"),
    CardsPage: page("cards-screen"),
    StartPage: page("start-screen"),
    NotFoundPage: page("not-found-screen"),
    SettingsPage: page("settings-screen"),
    PromptSettingsPage: page("prompts-screen"),
    NewChapterPage: page("new-chapter-screen"),
  };
});

function Where() {
  const { pathname, search } = useLocation();
  return <output data-testid="where">{`${pathname}${search}`}</output>;
}

function renderAt(path: string) {
  return render(
    <UIProvider linkAs={Link}>
      <MemoryRouter initialEntries={[path]}>
        <Where />
        <AppRoutes shell={<AppShell />} />
      </MemoryRouter>
    </UIProvider>,
  );
}

const where = () => screen.getByTestId("where").textContent;

function setDesktop(desktop: boolean) {
  window.matchMedia = ((query: string) => ({
    matches: desktop && query.includes("min-width: 1024px"),
    media: query,
    onchange: null,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia;
}

describe("AppShell (standalone frame)", () => {
  const original = window.matchMedia;
  beforeEach(() => {
    window.sessionStorage.clear();
    state.issue = false;
    setDesktop(false);
  });
  afterEach(() => {
    window.matchMedia = original;
  });

  it("draws brand, section tabs with the badge, the screen title and the page", () => {
    renderAt("/");
    expect(screen.getByRole("heading", { level: 1, name: "오늘" })).toBeInTheDocument();
    expect(document.title).toBe("오늘 · punisher");
    const tabs = screen.getByRole("navigation", { name: "punisher 섹션" });
    expect(within(within(tabs).getByRole("link", { name: /오늘/ })).getByText("3")).toBeInTheDocument();
    // 상단 바에 검색·설정 링크가 없다(설정은 페이지 동작)
    expect(screen.queryByRole("link", { name: "검색" })).toBeNull();
    // 섹션 첫 화면에는 ← 가 없다
    expect(screen.queryByRole("link", { name: "뒤로" })).toBeNull();
    expect(screen.getByText("today-screen")).toBeInTheDocument();
  });

  it("redirects /goals and /search with replace", () => {
    renderAt("/goals");
    expect(where()).toBe("/map?panel=goals");
  });

  it("keeps the query when /search moves to /history", () => {
    renderAt("/search?q=tcp");
    expect(where()).toBe("/history?q=tcp");
  });

  it("serves /start in the standalone app too", () => {
    renderAt("/start?concept=c1");
    expect(screen.getByText("start-screen")).toBeInTheDocument();
  });

  it("desktop: text buttons, 설정, and ⋯", () => {
    setDesktop(true);
    renderAt("/map");
    expect(screen.getByRole("link", { name: "새 챕터" })).toHaveAttribute("href", "/chapters/new");
    expect(screen.getByRole("link", { name: "설정" })).toHaveAttribute("href", "/settings");
    expect(screen.getByRole("button", { name: "동작 더 보기" })).toBeInTheDocument();
  });

  it("phone: one icon button, settings in ⋯", () => {
    renderAt("/map");
    const icon = screen.getByRole("link", { name: "새 챕터" });
    expect(icon).toHaveAttribute("href", "/chapters/new");
    expect(icon.textContent).toBe("");
    expect(screen.queryByRole("link", { name: "설정" })).toBeNull();
    expect(screen.getByRole("button", { name: "동작 더 보기" })).toBeInTheDocument();
  });

  it("has no settings action on the settings screen", () => {
    setDesktop(true);
    renderAt("/settings");
    expect(screen.getByRole("heading", { level: 1, name: "punisher 설정" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "설정" })).toBeNull();
    expect(screen.queryByRole("button", { name: "동작 더 보기" })).toBeNull();
  });

  it("← from a direct visit replaces with the section first screen", () => {
    renderAt("/chapters/new");
    fireEvent.click(screen.getByRole("link", { name: "뒤로" }));
    expect(where()).toBe("/map");
  });

  it("← after moving inside the app goes back", async () => {
    renderAt("/history");
    fireEvent.click(screen.getByRole("link", { name: "to-new-chapter" }));
    expect(where()).toBe("/chapters/new");
    await act(async () => { fireEvent.click(screen.getByRole("link", { name: "뒤로" })); });
    expect(where()).toBe("/history");
  });

  it("shows the Claude banner only on screens that generate", () => {
    state.issue = true;
    const first = renderAt("/");
    expect(screen.getByText("claude-banner")).toBeInTheDocument();
    first.unmount();
    renderAt("/map");
    expect(screen.queryByText("claude-banner")).toBeNull();
  });
});
