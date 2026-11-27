import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ConceptDetail, Graph, GraphNode } from "@/src/shared/api/punisher";

const node = (over: Partial<GraphNode> & Pick<GraphNode, "id" | "name">): GraphNode => ({
  area: "os", depth: 1, status: "active", state: "not_started", locked: false, frontier: false, unlocks: 0, ...over,
});

const graphNodes: GraphNode[] = [
  node({ id: "os.process", name: "프로세스", state: "in_progress" }),
  node({ id: "os.thread", name: "스레드", frontier: true }),
  node({ id: "os.paging", name: "페이징", depth: 2, locked: true }),
  node({ id: "db.index", name: "인덱스", area: "database", state: "completed" }),
];

const graph = (nodes: GraphNode[]): Graph => ({
  curriculumVersion: "v1",
  nodes,
  edges: [],
  coverage: [
    { area: "os", total: 4, completed: 1, known: 0, inProgress: 1, requested: 0, dismissed: 0 },
    { area: "database", total: 2, completed: 1, known: 0, inProgress: 0, requested: 0, dismissed: 0 },
  ],
  pendingRequests: [],
  chapters: [],
  foundationsHidden: false,
});

const detail = (id: string, over: Partial<ConceptDetail> = {}): ConceptDetail =>
  ({
    id, name: graphNodes.find((n) => n.id === id)?.name ?? id, area: "os", depth: 1, summary: "", practicalCases: null, docLinks: null,
    status: "active", origin: "seed", objectiveCount: 0, userState: "not_started", updatedAt: "", aliases: [], objectives: [],
    prerequisites: [{ id: "os.process", name: "프로세스", area: "os", kind: "prerequisite", userState: "in_progress" }],
    dependents: [], related: [], locked: false, ...over,
  }) as ConceptDetail;

const state = {
  nodes: graphNodes,
  lectures: [] as Array<{ id: string }>,
  userState: "not_started" as ConceptDetail["userState"],
  locked: false,
  dismissed: false,
};
const markKnown = vi.fn();
const undoKnown = vi.fn();
const createStudyRequest = vi.fn();
const restoreConcept = vi.fn();

const ok = <T,>(data: T) => ({ data, isLoading: false, isSuccess: true, error: null, refetch: vi.fn() });
const idle = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(async () => undefined), isPending: false, isSuccess: false, error: null });

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useGraph: () => ok(graph(state.nodes)),
  useGoals: () => ok([]),
  useGoalPath: () => ({ ...ok(undefined), data: undefined }),
  useMasteryBatch: () => ok([]),
  useToday: () => ok({ items: [] }),
  useConceptDetail: (id?: string) => (id ? ok(detail(id, { userState: state.userState, locked: state.locked, dismissed: state.dismissed })) : { ...ok(undefined), data: undefined }),
  useConceptLectures: () => ok(state.lectures),
  useNotes: () => ok([{ id: "n1" }]),
  useMyQuestions: () => ({ ...ok({ pages: [{ items: [{ id: "q1" }, { id: "q2" }] }] }), hasNextPage: false }),
  useInquiries: () => ok([]),
  useMarkKnown: () => ({ ...idle(), mutate: markKnown }),
  useUndoKnown: () => ({ ...idle(), mutate: undoKnown }),
  useCreateStudyRequest: () => ({ ...idle(), mutate: createStudyRequest }),
  useConceptDismiss: () => ({ ...idle(), mutate: restoreConcept }),
}));

// 시트 본문은 각자 테스트가 있다 — 여기서는 주소 → 시트 열림만 본다.
vi.mock("@/src/features/punisher/goal-manage", () => ({ GoalsManager: () => <p>목표 목록</p> }));
vi.mock("@/src/features/punisher/known-onboarding", () => ({
  KnownAreasSheet: ({ open, onClose }: { open: boolean; onClose: () => void }) =>
    open ? <div role="dialog" aria-label="이미 아는 개념 표시"><button onClick={onClose}>닫기</button></div> : null,
}));
vi.mock("../ui/MapCanvas", () => ({
  default: ({ onSelect }: { onSelect: (id: string) => void }) => <button onClick={() => onSelect("os.thread")}>그래프 노드 스레드</button>,
}));

const { KnowledgeMapWidget } = await import("../KnowledgeMapWidget");
const { useScreenChrome } = await import("@/src/shared/lib/screen-chrome");

function setSplit(split: boolean) {
  window.matchMedia = ((query: string) => ({
    matches: split && query.includes("min-width: 1024px"),
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
    onchange: null,
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia;
}

function Probe() {
  const location = useLocation();
  const chrome = useScreenChrome();
  return (
    <>
      <output data-testid="url">{`${location.pathname}${location.search}`}</output>
      <output data-testid="title">{chrome.title ?? ""}</output>
      <ul aria-label="페이지 동작">
        {(chrome.actions ?? []).map((a) => (
          <li key={a.id}>
            <button onClick={() => a.onSelect?.()} data-kind={a.kind ?? ""} data-href={a.href ?? ""}>{a.label}</button>
          </li>
        ))}
      </ul>
    </>
  );
}

function renderMap(url: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[url]}>
      <QueryClientProvider client={client}>
        <KnowledgeMapWidget />
        <Probe />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const url = () => screen.getByTestId("url").textContent;

beforeEach(() => {
  state.nodes = graphNodes;
  state.lectures = [];
  state.userState = "not_started";
  state.locked = false;
  state.dismissed = false;
  restoreConcept.mockReset();
  Object.defineProperty(window, "scrollTo", { value: () => undefined, writable: true });
  setSplit(true);
});

describe("지도 페이지 동작", () => {
  it("hands the shell 새 챕터 (default, link) then 목표 관리 · 이미 아는 개념 표시 without a kind", async () => {
    renderMap("/map");
    const actions = await screen.findByRole("list", { name: "페이지 동작" });
    const items = within(actions).getAllByRole("button");
    expect(items.map((b) => b.textContent)).toEqual(["새 챕터", "목표 관리", "이미 아는 개념 표시"]);
    expect(items[0]).toHaveAttribute("data-kind", "default");
    expect(items[0]).toHaveAttribute("data-href", "/chapters/new");
    expect(items[1]).toHaveAttribute("data-kind", "");
    expect(items[2]).toHaveAttribute("data-kind", "");
  });

  it("opens the sheets by ?panel= and closes them again", async () => {
    renderMap("/map?view=list");
    fireEvent.click(await screen.findByRole("button", { name: "목표 관리" }));
    expect(url()).toBe("/map?view=list&panel=goals");
    expect(await screen.findByText("목표 목록")).toBeInTheDocument();

    // 대화상자가 열려 있는 동안 뒤의 페이지 동작은 가려진다(aria-hidden).
    fireEvent.click(screen.getByRole("button", { name: "이미 아는 개념 표시", hidden: true }));
    expect(url()).toBe("/map?view=list&panel=known");
    fireEvent.click(await screen.findByRole("button", { name: "닫기" }));
    // 기록이 없으면(직접 들어온 주소) 시트 없는 주소로 교체한다.
    await waitFor(() => expect(url()).toBe("/map?view=list"));
  });

  it("opens a sheet that the address already names", async () => {
    renderMap("/map?panel=known");
    expect(await screen.findByRole("dialog", { name: "이미 아는 개념 표시" })).toBeInTheDocument();
  });
});

describe("지도 툴바와 본문", () => {
  it("lists areas with their progress percent and has no 새 챕터·⋯ in the toolbar", async () => {
    renderMap("/map");
    await screen.findByRole("button", { name: "범례" });
    expect(screen.getByRole("radio", { name: "목록" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "그래프" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "지도 메뉴" })).toBeNull();
    expect(screen.queryByText("다음에 공부하면 좋은 것")).toBeNull();
    expect(screen.queryByText("분류 중인 학습 요청")).toBeNull();
    expect(screen.queryByText("영역별 커버리지")).toBeNull();
  });

  it("lists 필요 없음 in the legend next to the other states", async () => {
    renderMap("/map");
    fireEvent.click(await screen.findByRole("button", { name: "범례" }));
    const legend = await screen.findByRole("dialog", { name: "범례" });
    expect(within(legend).getByText("필요 없음")).toBeInTheDocument();
    expect(within(legend).getByText("추천에서 뺐어요")).toBeInTheDocument();
    expect(within(legend).getAllByRole("listitem")).toHaveLength(8);
  });

  it("shows the graph and the empty detail panel on a wide screen, the list on a phone", async () => {
    renderMap("/map");
    expect(await screen.findByRole("button", { name: "그래프 노드 스레드" })).toBeInTheDocument();
    expect(screen.getByText("지도에서 개념을 고르세요")).toBeInTheDocument();
  });

  it("starts in the list on a phone, grouped by area", async () => {
    setSplit(false);
    renderMap("/map");
    expect(await screen.findByRole("heading", { name: "운영체제" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "데이터베이스" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "그래프 노드 스레드" })).toBeNull();
  });

  it("shows 아직 지도가 비었어요 with a 새 챕터 열기 link when there are no concepts", async () => {
    state.nodes = [];
    renderMap("/map");
    expect(await screen.findByText("아직 지도가 비었어요")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "새 챕터 열기" })).toHaveAttribute("href", "/chapters/new");
  });
});

describe("개념 상세 ?node=", () => {
  it("opens with push the first time and replaces when another concept is picked", async () => {
    renderMap("/map");
    fireEvent.click(await screen.findByRole("button", { name: "그래프 노드 스레드" }));
    expect(url()).toBe("/map?node=os.thread");
    expect(await screen.findByRole("heading", { name: "스레드" })).toBeInTheDocument();
    // 선행 개념으로 넘어가면 같은 상세 안에서 교체
    fireEvent.click(screen.getByRole("button", { name: "프로세스" }));
    expect(url()).toBe("/map?node=os.process");
  });

  it("reads the old ?concept= too, and × closes the panel", async () => {
    renderMap("/map?concept=os.thread");
    expect(await screen.findByRole("heading", { name: "스레드" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "개념 상세 닫기" }));
    await waitFor(() => expect(url()).toBe("/map"));
    expect(screen.getByText("지도에서 개념을 고르세요")).toBeInTheDocument();
  });

  it("shows area · state, 강의 시작 when there is no lecture, and the 질문·메모 link to the history tab", async () => {
    renderMap("/map?node=os.thread");
    const panel = await screen.findByRole("region", { name: "스레드" });
    expect(within(panel).getByText("운영체제 · 추천")).toBeInTheDocument();
    expect(within(panel).getByRole("link", { name: "강의 시작" })).toHaveAttribute("href", "/start?concept=os.thread");
    expect(within(panel).getByRole("link", { name: "질문·메모 3개 보기" })).toHaveAttribute("href", `/history?tab=notes&q=${encodeURIComponent("스레드")}`);
    // 질문·메모 본문은 여기서 그리지 않는다.
    expect(screen.queryByText("내 질문")).toBeNull();
    expect(screen.queryByText("내 메모")).toBeNull();
  });

  it("says 강의 열기 when a lecture exists, and disables the button for a locked concept without one", async () => {
    state.lectures = [{ id: "l1" }];
    const { unmount } = renderMap("/map?node=os.thread");
    expect(await screen.findByRole("link", { name: "강의 열기" })).toHaveAttribute("href", "/start?concept=os.thread");
    unmount();
    state.lectures = [];
    state.locked = true;
    renderMap("/map?node=os.paging");
    expect(await screen.findByRole("button", { name: "앞 개념을 먼저 배워요" })).toBeDisabled();
  });

  it("saves 이미 앎 right away from the switch", async () => {
    renderMap("/map?node=os.thread");
    fireEvent.click(await screen.findByRole("switch", { name: "스레드 이미 앎" }));
    expect(markKnown).toHaveBeenCalledWith("os.thread", expect.anything());
  });

  it("shows 필요 없음 and [다시 추천받기] for a concept taken off the picks, and puts it back (dismissed=false)", async () => {
    state.dismissed = true;
    state.nodes = graphNodes.map((n) => (n.id === "os.thread" ? { ...n, frontier: false, dismissed: true } : n));
    renderMap("/map?node=os.thread");
    const panel = await screen.findByRole("region", { name: "스레드" });
    expect(within(panel).getByText("운영체제 · 필요 없음")).toBeInTheDocument();
    // 빼는 동작은 지도에 없다 — 되돌리는 버튼만.
    expect(within(panel).queryByRole("button", { name: /필요 없음으로|추천에서 빼/ })).toBeNull();
    restoreConcept.mockImplementation((_vars, options: { onSuccess?: () => void }) => options.onSuccess?.());
    fireEvent.click(within(panel).getByRole("button", { name: "다시 추천받기" }));
    expect(restoreConcept).toHaveBeenCalledWith({ conceptId: "os.thread", dismissed: false }, expect.anything());
    expect(await screen.findByText("추천에 다시 넣었어요")).toBeInTheDocument();
  });

  it("has no [다시 추천받기] for a concept that was never taken off", async () => {
    renderMap("/map?node=os.thread");
    const panel = await screen.findByRole("region", { name: "스레드" });
    expect(within(panel).getByText("운영체제 · 추천")).toBeInTheDocument();
    expect(within(panel).queryByRole("button", { name: "다시 추천받기" })).toBeNull();
  });

  it("keeps 배움 for a finished concept even if it was dismissed earlier — no restore button", async () => {
    state.dismissed = true;
    state.userState = "completed";
    state.nodes = graphNodes.map((n) => (n.id === "os.thread" ? { ...n, state: "completed", frontier: false, dismissed: true } : n));
    renderMap("/map?node=os.thread");
    const panel = await screen.findByRole("region", { name: "스레드" });
    expect(within(panel).getByText("운영체제 · 배움")).toBeInTheDocument();
    expect(within(panel).queryByRole("button", { name: "다시 추천받기" })).toBeNull();
  });

  it("puts the concept on the study list with 나중에 공부", async () => {
    renderMap("/map?node=os.thread");
    fireEvent.click(await screen.findByRole("button", { name: "나중에 공부" }));
    expect(createStudyRequest).toHaveBeenCalledWith("스레드", expect.anything());
  });

  it("is a full screen titled with the concept name on a phone, without the toolbar or the page actions", async () => {
    setSplit(false);
    renderMap("/map?node=os.thread");
    await screen.findByRole("link", { name: "강의 시작" });
    expect(screen.getByTestId("title")).toHaveTextContent("스레드");
    expect(screen.queryByRole("button", { name: "범례" })).toBeNull();
    expect(screen.queryByRole("button", { name: "새 챕터" })).toBeNull();
    expect(screen.queryByRole("button", { name: "개념 상세 닫기" })).toBeNull();
  });
});
