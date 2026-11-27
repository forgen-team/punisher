import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { Inquiry, MyQuestion, Note, SearchHit } from "@/src/shared/api/punisher";

const inquiry = { id: "i1", question: "캐시 무효화는 언제 하나요", status: "answered", createdAt: "2026-11-20T00:00:00Z", followupCount: 0, conceptIds: [], proposedConceptIds: [], model: "", promptVersion: "", updatedAt: "" } as Inquiry;
const lq = { id: "q1", lectureId: "l1", anchor: "intro", kind: "question", question: "TTL 이 뭔가요", status: "answered", createdAt: "2026-11-10T00:00:00Z", lectureTitle: "HTTP 캐시", conceptName: "캐시" } as MyQuestion;
const memo = { id: "n1", lectureId: "l1", anchor: "body", kind: "note", quote: "", body: "ETag 정리", createdAt: "2026-11-15T00:00:00Z", updatedAt: "" } as Note;
const hits: SearchHit[] = [
  { kind: "note", refId: "n1", lectureId: "l1", anchor: "body", title: "ETag 정리", sectionTitle: "", snippet: "…[[캐시]] 검증…" },
  { kind: "inquiry", refId: "i1", lectureId: "", anchor: "", title: "캐시 무효화는 언제 하나요", sectionTitle: "", snippet: "[[캐시]]" },
  { kind: "lecture", refId: "l1", lectureId: "l1", anchor: "s1", title: "HTTP 캐시", sectionTitle: "개요", snippet: "[[캐시]] 개요" },
];

const api = {
  listInquiries: vi.fn(async () => [inquiry]),
  listMyQuestions: vi.fn(async () => ({ items: [lq], nextCursor: "" })),
  listNotes: vi.fn(async () => [memo]),
  listConcepts: vi.fn(async () => []),
  completedAreas: vi.fn(async () => [{ area: "network", count: 2 }]),
  listCompleted: vi.fn(async () => ({ items: [], nextCursor: "" })),
  listCheckHistory: vi.fn(async () => ({ sessions: [], nextCursor: "" })),
};
const studyApi = { search: vi.fn(async (q: string): Promise<SearchHit[]> => (q === "캐시" ? hits : [])) };

vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return {
    ...actual,
    punisherApi: new Proxy(actual.punisherApi, { get: (target, key) => (key in api ? api[key as keyof typeof api] : Reflect.get(target, key)) }),
    studyApi: new Proxy(actual.studyApi, { get: (target, key) => (key in studyApi ? studyApi[key as keyof typeof studyApi] : Reflect.get(target, key)) }),
  };
});

const { HistoryWidget } = await import("../HistoryWidget");

function Where() {
  const { pathname, search } = useLocation();
  return <output data-testid="where">{pathname + search}</output>;
}

function renderAt(url: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(
    <MemoryRouter initialEntries={[url]}>
      <QueryClientProvider client={client}>
        <HistoryWidget />
        <Where />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const where = () => screen.getByTestId("where").textContent;

describe("기록 화면", () => {
  it("shows three tabs and one search box", async () => {
    renderAt("/history");
    const tabs = screen.getByRole("navigation", { name: "기록 종류" });
    expect(Array.from(tabs.querySelectorAll("a")).map((a) => a.textContent)).toEqual(["학습", "복습·통계", "질문·메모"]);
    expect(screen.getAllByRole("searchbox")).toHaveLength(1);
    expect(screen.getByPlaceholderText("강의·개념·질문·메모에서 찾기")).toBeInTheDocument();
    expect(await screen.findByText("아직 들은 강의가 없어요")).toBeInTheDocument();
  });

  it("maps an old tab value to the new tab and replaces the address", async () => {
    renderAt("/history?tab=questions");
    await waitFor(() => expect(where()).toBe("/history?tab=notes&kind=lq"));
    expect(await screen.findByText("TTL 이 뭔가요")).toBeInTheDocument();
    expect(screen.queryByText("ETag 정리")).toBeNull();
    expect(screen.getByRole("checkbox", { name: "강의 중 질문" })).toBeChecked();
  });

  it("merges questions and notes into one list with a kind chip", async () => {
    renderAt("/history?tab=notes");
    expect(await screen.findByText("TTL 이 뭔가요")).toBeInTheDocument();
    expect(screen.getByText("ETag 정리")).toBeInTheDocument();
    expect(screen.getByText(/^질문 해설 · /)).toBeInTheDocument();
    expect(screen.getByText("강의 중 질문 · HTTP 캐시 · 2026. 11. 10")).toBeInTheDocument();
    expect(screen.getByText("메모 · HTTP 캐시 · 2026. 11. 15")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "메모" }));
    await waitFor(() => expect(where()).toBe("/history?tab=notes&kind=memo"));
    expect(screen.queryByText("TTL 이 뭔가요")).toBeNull();
    expect(screen.getByText("ETag 정리").closest("a")).toHaveAttribute("href", "/lectures/l1?at=body&panel=notes");
  });

  it("puts each tab's result count next to its name when a search term is in the address", async () => {
    renderAt("/history?tab=notes&q=캐시");
    const tabs = screen.getByRole("navigation", { name: "기록 종류" });
    await waitFor(() => expect(tabs.textContent).toBe("학습2복습·통계0질문·메모2"));
    const rows = await screen.findAllByRole("listitem");
    expect(rows).toHaveLength(2); // 강의 본문 결과는 학습 탭 몫
  });

  it("says nothing matches and offers to clear the search", async () => {
    renderAt("/history?tab=notes&q=없는말");
    expect(await screen.findByText("‘없는말’에 맞는 기록이 없어요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "검색어 지우기" }));
    await waitFor(() => expect(where()).toBe("/history?tab=notes"));
  });

  it("switches tabs with replace and keeps the search term", async () => {
    renderAt("/history?q=캐시");
    fireEvent.click(screen.getByRole("link", { name: /복습·통계/ }));
    await waitFor(() => expect(where()).toBe(`/history?tab=review&q=${encodeURIComponent("캐시")}`));
  });
});
