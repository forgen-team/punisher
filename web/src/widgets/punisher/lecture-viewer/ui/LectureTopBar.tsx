
import { useEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { Link } from "@/src/shared/lib/router";
import { useRouter } from "@/src/shared/lib/router";
import { canGoBack } from "@/src/shared/lib/in-app-history";
import { getAppElementById } from "@/src/shared/lib/dom-root";
import { isWitems } from "@/src/shared/config/runtime";
import { useCollapsingAppHeader } from "@/src/shared/lib/appHeaderCollapse";
import { ArrowLeft, ListOrdered, MessagesSquare } from "lucide-react";
import { Button, IconButton, Menu, type MenuItemDef } from "@forgen-team/ui";
import { cn } from "@/src/shared/lib/cn";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

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
      const heading = getAppElementById(LECTURE_TITLE_ID, bar.current);
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
  /** 질문 아이콘 위 개수 표시(스레드 수) */
  assistBadge?: ReactNode;
  menuItems?: MenuItemDef[];
  /**
   * ⋯ 대신 글자 버튼으로 보일 메뉴 이름(강의: "학습 상태"/"완료함"). 셸 상단 바의 ⋯(페이지 동작) 바로 아래라
   * 두 ⋯ 가 붙어 헷갈리지 않게 한다. short 는 아주 좁은 폭(400px 미만)용.
   */
  menuText?: { full: string; short: string };
}

/**
 * 강의 상단 바(고정): 뒤로 · 제목(한 줄) · 목차 · 질문/메모 · ⋯.
 * 모바일에선 셸 상단 바 아래(--app-header-offset), 데스크톱에선 0 에 붙는다.
 * 제목 중복 방지: 본문 큰 제목이 보이는 동안 여기 제목은 숨기고(자리는 유지 → 레이아웃 시프트 없음), 스크롤로 지나가면 나타난다.
 * 뒤로: 단독판 셸 상단 바에는 [←]가 없으므로 모든 폭에서 보인다(witems 는 lg 이상만 — docs/SYNC.md).
 * 질문·메모를 하단 FAB이 아니라 여기 두어 하단 탭 바와 겹치지 않게 한다.
 * 휴대폰: 아래로 읽는 동안 셸 상단 바가 숨고 이 바만 맨 위에 남는다(위로 되돌아가면 셸 바가 다시 보인다).
 */
export function LectureTopBar({ title, backHref = "/", onOpenToc, onOpenAssist, assistBadge, menuItems, menuText }: LectureTopBarProps) {
  const router = useRouter();
  const t = useT(messages);
  const barRef = useRef<HTMLDivElement>(null);
  const titleAway = useTitleScrolledAway(barRef, title);
  useCollapsingAppHeader();
  // Witems: ← 와 화면 제목은 셸 상단 바가 그린다 — 이 바에는 목차·질문/메모·⋯ 만 둔다.
  const inWitems = isWitems();
  return (
    <div ref={barRef} data-lecture-topbar="" className="sticky top-[var(--app-header-offset)] z-nav -mx-4 flex h-12 transition-[top] duration-base ease-standard items-center gap-1 border-b border-line bg-canvas px-2 fold:-mx-6 fold:px-4 lg:-mx-8 lg:px-6">
      {inWitems ? null : <Button asChild variant="ghost" aria-label={t.back}><Link
        href={backHref}
        onClick={(event) => {
          // 앱 안에서 들어왔으면 들어온 곳(지도·기록 등)으로, 직접 들어왔으면 강의 화면으로.
          if (canGoBack()) {
            event.preventDefault();
            router.back();
          }
        }}
        // 단독판: 셸 상단 바에 ← 가 없어 모든 폭에서 보인다(witems 는 lg 이상만 — docs/SYNC.md).
        aria-label={t.back}>
        <ArrowLeft aria-hidden="true" />
      </Link></Button>}
      <p
        className={cn("min-w-0 flex-1 truncate text-sm font-semibold text-fg transition-opacity duration-150", titleAway ? "opacity-100" : "opacity-0")}
        aria-hidden="true"
      >
        {inWitems ? null : title}
      </p>
      {onOpenToc && (
        <IconButton aria-label={t.toc} onClick={onOpenToc} className="lg:hidden" icon={<ListOrdered aria-hidden="true" />} />
      )}
      {onOpenAssist && (
        // ui IconButton 은 아이콘만 받는다 — 개수 표시는 감싼 상자에 겹쳐 둔다.
        <span className="relative inline-flex xl:hidden">
          <IconButton aria-label={t.questionsAndNotes} onClick={onOpenAssist} icon={<MessagesSquare aria-hidden="true" />} />
          {assistBadge}
        </span>
      )}
      {menuItems && menuItems.length > 0 && (
        <Menu
          items={menuItems}
          label={t.lectureMenu}
          trigger={menuText ? (
            <Button variant="ghost" size="sm">
              <span className="hidden min-[400px]:inline">{menuText.full}</span>
              <span className="min-[400px]:hidden">{menuText.short}</span>
            </Button>
          ) : undefined}
        />
      )}
    </div>
  );
}
