import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { Inquiry } from "@/src/shared/api/punisher";
import type { PageAction } from "@/src/shared/lib/screen-chrome";

const setRead = { mutate: vi.fn(), isPending: false, error: null };
const regenerate = { mutate: vi.fn(), isPending: false, error: null };
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
  Link: ({ href, children }: { href: string; children?: ReactNode }) => <a href={href}>{children}</a>,
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
  useActiveSection: () => "",
}));
vi.mock("../ui/QuoteAsk", () => ({ QuoteAsk: () => null }));

const { InquiryView } = await import("../InquiryView");

describe("질문 해설", () => {
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
});
