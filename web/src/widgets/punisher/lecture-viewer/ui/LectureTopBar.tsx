
import { useEffect, useRef, useState, type RefObject } from "react";
import { Link } from "@/src/shared/lib/router";
import { ArrowLeft, ListOrdered, MessagesSquare } from "lucide-react";
import { IconButton, Menu, cn, iconButtonVariants, type MenuItem } from "@/src/shared/ui/kit";

/** 본문 강의 제목(h1) id — 이 제목이 화면 위로 지나가면 상단 바에 작게 나타난다. */
export const LECTURE_TITLE_ID = "lecture-title";

/**
 * 본문 제목(h1)이 이 바 아래로 지나갔는가 — 스크롤·크기 변경 때 바의 아래 끝과 제목의 아래 끝을 비교한다.
 * 제목이 없으면(즉석 강의 생성 중 등) 항상 보인다.
 */
function useTitleScrolledAway(bar: RefObject<HTMLElement>, title: string): boolean {
  const [away, setAway] = useState(false);
  useEffect(() => {
    const update = () => {
      const heading = document.getElementById(LECTURE_TITLE_ID);
      const barEl = bar.current;
      const next = !heading || !barEl ? true : heading.getBoundingClientRect().bottom <= barEl.getBoundingClientRect().bottom;
      setAway((prev) => (prev === next ? prev : next));
    };
    update();
    window.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update);
    return () => {
      window.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, [bar, title]);
  return away;
}

interface LectureTopBarProps {
  title: string;
  backHref?: string;
  /** 목차 시트 열기 (lg 이상은 왼쪽 목차가 항상 보이므로 숨김) */
  onOpenToc?: () => void;
  /** 질문·메모 시트 열기 */
  onOpenAssist?: () => void;
  menuItems?: MenuItem[];
}

/**
 * 강의 상단 바(고정): 뒤로 · 제목(한 줄) · 목차 · 질문/메모 · ⋯.
 * 모바일에선 셸 상단 바 아래(--app-header-offset), 데스크톱에선 0 에 붙는다.
 * 제목 중복 방지: 본문 큰 제목이 보이는 동안 여기 제목은 숨기고(자리는 유지 → 레이아웃 시프트 없음), 스크롤로 지나가면 나타난다.
 * 뒤로: 단독판 셸 상단 바에는 [←]가 없으므로 모든 폭에서 보인다(witems 는 lg 이상만 — docs/SYNC.md).
 * 질문·메모를 하단 FAB이 아니라 여기 두어 하단 탭 바와 겹치지 않게 한다.
 */
export function LectureTopBar({ title, backHref = "/", onOpenToc, onOpenAssist, menuItems }: LectureTopBarProps) {
  const barRef = useRef<HTMLDivElement>(null);
  const titleAway = useTitleScrolledAway(barRef, title);
  return (
    <div ref={barRef} className="sticky top-[var(--app-header-offset)] z-nav -mx-4 flex h-12 items-center gap-1 border-b border-line bg-canvas px-2 md:-mx-6 md:px-4">
      <Link href={backHref} aria-label="강의 목록으로" className={iconButtonVariants({ variant: "ghost" })}>
        <ArrowLeft aria-hidden="true" />
      </Link>
      <p
        className={cn("min-w-0 flex-1 truncate text-sm font-semibold text-fg transition-opacity duration-150", titleAway ? "opacity-100" : "opacity-0")}
        aria-hidden="true"
      >
        {title}
      </p>
      {onOpenToc && (
        <IconButton aria-label="목차" onClick={onOpenToc} className="lg:hidden">
          <ListOrdered aria-hidden="true" />
        </IconButton>
      )}
      {onOpenAssist && (
        <IconButton aria-label="질문과 메모" onClick={onOpenAssist}>
          <MessagesSquare aria-hidden="true" />
        </IconButton>
      )}
      {menuItems && menuItems.length > 0 && <Menu items={menuItems} label="강의 메뉴" />}
    </div>
  );
}
