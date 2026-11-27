import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation } from "react-router-dom";
import { FeedbackProvider, UIProvider } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { AppRoutes } from "../App";
import { AppShell } from "../AppShell";

const state = vi.hoisted(() => ({ issue: false, select: vi.fn(), startReview: vi.fn() }));

vi.mock("@/src/entities/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/entities/punisher")>();
  return {
    lectureHref: actual.lectureHref,
    startLectureHref: actual.startLectureHref,
    LocaleSync: () => null,
    useActivity: () => ({ data: { restToday: false, today: { reviewsDue: 2, cardsDue: 0 } } }),
    useToday: () => ({ data: { items: [{ kind: "goal_path", conceptId: "c1", conceptName: "TCP", title: "TCP" }], exhausted: false, unchecked: [{ lectureId: "l1" }] } }),
    useStartReview: () => ({ mutate: state.startReview }),
  };
});

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
      <FeedbackProvider>
        <MemoryRouter initialEntries={[path]}>
          <Where />
          <AppRoutes shell={<AppShell />} />
        </MemoryRouter>
      </FeedbackProvider>
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

  it("draws the AppFrame: brand, ⌘K, section tabs with the badge and current, the screen title and the page", () => {
    renderAt("/");
    expect(screen.getByRole("link", { name: "punisher 처음 화면" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("button", { name: /^명령/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "오늘" })).toBeInTheDocument();
    expect(document.title).toBe("오늘 · punisher");
    const tabs = screen.getByRole("navigation", { name: "punisher 섹션" });
    const today = within(tabs).getByRole("link", { name: /오늘/ });
    expect(within(today).getByText("3")).toBeInTheDocument();
    expect(today).toHaveAttribute("aria-current", "page");
    // 오늘 화면만 틀 폭 today
    expect(document.querySelector(".fg-frame--today")).not.toBeNull();
    // 상단 바에 검색·설정 링크가 없다(설정은 페이지 동작)
    expect(screen.queryByRole("link", { name: "검색" })).toBeNull();
    // 섹션 첫 화면에는 ← 가 없다
    expect(screen.queryByRole("link", { name: "뒤로" })).toBeNull();
    expect(screen.getByText("today-screen")).toBeInTheDocument();
  });

  it("marks the section of a sub-screen and uses the default frame width", () => {
    renderAt("/chapters/new");
    const tabs = screen.getByRole("navigation", { name: "punisher 섹션" });
    expect(within(tabs).getByRole("link", { name: "지도" })).toHaveAttribute("aria-current", "page");
    expect(document.querySelector(".fg-frame--default")).not.toBeNull();
  });

  it("⌘K opens the palette with the shell's command list; a link command moves", () => {
    renderAt("/map");
    expect(screen.queryByRole("dialog")).toBeNull();
    fireEvent.keyDown(window, { key: "k", code: "KeyK", metaKey: true });
    const palette = screen.getByRole("dialog");
    const options = within(palette).getAllByRole("option").map((o) => o.textContent ?? "");
    for (const label of ["punisher 오늘", "다음 강의", "복습 시작", "punisher 설정"]) {
      expect(options.some((text) => text.includes(label))).toBe(true);
    }
    fireEvent.click(within(palette).getByRole("option", { name: /punisher 설정/ }));
    expect(where()).toBe("/settings");
  });

  it("the ⌘K button opens the palette, and 복습 시작 runs its action", () => {
    renderAt("/");
    fireEvent.click(screen.getByRole("button", { name: /^명령/ }));
    fireEvent.click(within(screen.getByRole("dialog")).getByRole("option", { name: /복습 시작/ }));
    expect(state.startReview).toHaveBeenCalledTimes(1);
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

  it("⋯ puts the danger action last, after a separator", async () => {
    setDesktop(true);
    renderAt("/map");
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "동작 더 보기" })); });
    const menu = await screen.findByRole("menu");
    const items = within(menu).getAllByRole("menuitem");
    expect(items.map((i) => i.textContent)).toEqual(["목표 관리", "지우기"]);
    expect(within(menu).getAllByRole("separator")).toHaveLength(1);
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
