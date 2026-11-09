
import { Link } from "@/src/shared/lib/router";
import { ArrowLeft, ListOrdered, MessagesSquare } from "lucide-react";
import { IconButton, Menu, iconButtonVariants, type MenuItem } from "@/src/shared/ui/kit";

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
 * 질문·메모를 하단 FAB이 아니라 여기 두어 하단 탭 바와 겹치지 않게 한다.
 */
export function LectureTopBar({ title, backHref = "/", onOpenToc, onOpenAssist, menuItems }: LectureTopBarProps) {
  return (
    <div className="sticky top-[var(--app-header-offset)] z-nav -mx-4 flex h-12 items-center gap-1 border-b border-line bg-canvas px-2 md:-mx-6 md:px-4">
      <Link href={backHref} aria-label="강의 목록으로" className={iconButtonVariants({ variant: "ghost" })}>
        <ArrowLeft aria-hidden="true" />
      </Link>
      <p className="min-w-0 flex-1 truncate text-sm font-semibold text-fg" aria-hidden="true">
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
