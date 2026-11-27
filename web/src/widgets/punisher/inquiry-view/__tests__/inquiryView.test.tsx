import { fireEvent, render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Inquiry } from "@/src/shared/api/punisher";
import type { PageAction } from "@/src/shared/lib/screen-chrome";

const setRead = { mutate: vi.fn(), isPending: false, error: null };
const regenerate = { mutate: vi.fn(), isPending: false, error: null };
const router = { push: vi.fn(), replace: vi.fn(), back: vi.fn() };
let search = "";
let wide = false;
let pageActions: PageAction[] | null = null;
let titleSet: string | null | undefined;

const base: Inquiry = {
  id: "i1", question: "Redis 와 HTTP 캐시는 언제 나눠 쓰나요?", status: "answered",
  content: { title: "캐시 층 나누기", sections: [{ anchor: "s1", id: "s1", kind: "summary", title: "짧은 답", minutes: 3, blocks: [] } as never] },
  conceptIds: ["c2"], proposedConceptIds: [], concepts: [{ id: "c2", name: "HTTP 캐시", area: "web", kind: "related", userState: "learning" }],
  model: "", promptVersion: "", createdAt: "", updatedAt: "", answeredAt: "2026-11-27T09:00:00+09:00", followupCount: 0, followups: [],
};
let inquiry: Inquiry = base;

vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => router,
  usePathname: () => "/inquiries/i1",
  useSearchParams: () => new URLSearchParams(search),
  Link: ({ href, children }: { href: string; children?: ReactNode }) => <a href={href}>{children}</a>,
}));
vi.mock("@forgen-team/ui", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@forgen-team/ui")>()),
  useMediaQuery: (query: string) => (query === "(min-width: 1024px)" ? wide : false),
}));
vi.mock("@/src/shared/lib/screen-chrome", () => ({
  usePageActions: (actions: PageAction[] | null) => { pageActions = actions; },
  useScreenTitle: (title: string | null | undefined) => { titleSet = title; },
}));
vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useInquiry: () => ({ data: inquiry, error: null, refetch: vi.fn() }),
  useRegenerateInquiry: () => regenerate,
  useSetInquiryRead: () => setRead,
  useRetryFollowup: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useCreateFollowup: () => ({ mutate: vi.fn(), isPending: false, error: null }),
}));
vi.mock("@/src/features/punisher/instant-start", () => ({ useLaunchInstant: () => ({ launch: vi.fn(), isPending: false, error: null }) }));
vi.mock("../../lecture-viewer", () => ({
  LectureLayout: ({ children }: { children: ReactNode }) => <main>{children}</main>,
  LectureBody: () => <p>해설 본문</p>,
  LectureAssistPlaceholder: ({ children }: { children: ReactNode }) => <p>{children}</p>,
  SectionToc: ({ sections, onNavigate }: { sections: { anchor: string; title: string }[]; onNavigate?: (scroll: () => void) => void }) => (
    <nav aria-label="강의 목차">
      {sections.map((s) => <a key={s.anchor} href={`?at=${s.anchor}`} onClick={(e) => { e.preventDefault(); onNavigate?.(() => undefined); }}>{s.title}</a>)}
    </nav>
  ),
  useActiveSection: () => "",
}));
vi.mock("../ui/QuoteAsk", () => ({ QuoteAsk: () => null }));

const { InquiryView } = await import("../InquiryView");

describe("질문 해설", () => {
  beforeEach(() => {
    search = "";
    wide = false;
    router.push.mockClear();
    router.replace.mockClear();
    router.back.mockClear();
  });

  it("메타 줄, 끝 마무리 줄 [읽음으로 표시][이어서 묻기], 본문 글자 메뉴 없음", () => {
    inquiry = base;
    window.HTMLElement.prototype.scrollIntoView = vi.fn();
    render(<InquiryView inquiryId="i1" />);
    expect(titleSet).toBe("캐시 층 나누기");
    expect(screen.getByText("질문 해설 · 11월 27일 · 약 3분")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
    expect(screen.queryByRole("link", { name: "학습 홈으로" })).toBeNull();
    expect(screen.queryByRole("button", { name: /다시 만들기|관리/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "읽음으로 표시" }));
    expect(setRead.mutate).toHaveBeenCalledWith({ id: "i1", read: true });
    fireEvent.click(screen.getByRole("button", { name: "이어서 묻기" }));
    expect(screen.getByRole("textbox", { name: "이어서 물어볼 내용" })).toHaveFocus();
  });

  it("읽은 해설은 '읽음으로 표시했어요' 와 되돌리는 버튼", () => {
    inquiry = { ...base, readAt: "2026-11-27T10:00:00+09:00" };
    render(<InquiryView inquiryId="i1" />);
    expect(screen.getByText("읽음으로 표시했어요")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "읽음으로 표시" })).toBeNull();
    expect(screen.getByRole("button", { name: "목록에 다시 두기" })).toBeInTheDocument();
  });

  it("페이지 동작은 [다시 만들기][지도에서 보기](/map?node=개념)", () => {
    inquiry = base;
    render(<InquiryView inquiryId="i1" />);
    expect(pageActions?.map((a) => a.label)).toEqual(["다시 만들기", "지도에서 보기"]);
    expect(pageActions?.[1]?.href).toBe("/map?node=c2");
  });

  it("쓰는 중에는 다시 만들기를 내지 않는다", () => {
    inquiry = { ...base, status: "pending" };
    render(<InquiryView inquiryId="i1" />);
    expect(pageActions?.map((a) => a.label)).toEqual(["지도에서 보기"]);
  });

  it("폰: 본문 위 고정 바에 [목차] 하나, 누르면 ?panel=toc 로 push", () => {
    inquiry = base;
    render(<InquiryView inquiryId="i1" />);
    const bar = document.querySelector("[data-lecture-readbar]") as HTMLElement;
    expect(within(bar).getAllByRole("button").map((b) => b.textContent)).toEqual(["목차"]);
    expect(screen.queryByRole("dialog")).toBeNull();
    fireEvent.click(within(bar).getByRole("button", { name: "목차" }));
    expect(router.push).toHaveBeenCalledWith("/inquiries/i1?panel=toc", expect.objectContaining({ scroll: false }));
  });

  it("폰: ?panel=toc 로 들어오면 목차 시트가 열리고, 닫으면 panel 만 지운다", () => {
    inquiry = base;
    search = "panel=toc&foo=1";
    render(<InquiryView inquiryId="i1" />);
    const sheet = screen.getByRole("dialog", { name: "목차" });
    expect(within(sheet).getByRole("link", { name: "짧은 답" })).toBeInTheDocument();
    fireEvent.click(within(sheet).getByRole("button", { name: /닫기/ }));
    expect(router.replace).toHaveBeenCalledWith("/inquiries/i1?foo=1", expect.anything());
  });

  it("폰: 이 화면에서 연 시트는 뒤로가기로 닫힌다", () => {
    inquiry = base;
    const view = render(<InquiryView inquiryId="i1" />);
    fireEvent.click(screen.getByRole("button", { name: "목차" }));
    search = "panel=toc";
    view.rerender(<InquiryView inquiryId="i1" />);
    fireEvent.click(within(screen.getByRole("dialog", { name: "목차" })).getByRole("button", { name: /닫기/ }));
    expect(router.back).toHaveBeenCalledTimes(1);
    expect(router.replace).not.toHaveBeenCalled();
  });

  it("쓰는 중(섹션 없음)에는 바 자리는 두고 [목차] 는 눌리지 않는다", () => {
    inquiry = { ...base, status: "pending", content: undefined };
    render(<InquiryView inquiryId="i1" />);
    expect(screen.getByRole("button", { name: "목차" })).toBeDisabled();
  });

  it("보조 열이 있는 넓은 화면: 고정 바도 시트도 없다(?panel=toc 여도)", () => {
    inquiry = base;
    wide = true;
    search = "panel=toc";
    render(<InquiryView inquiryId="i1" />);
    expect(document.querySelector("[data-lecture-readbar]")).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
