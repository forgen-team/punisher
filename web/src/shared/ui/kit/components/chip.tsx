import * as React from "react";
import { cn, focusRing } from "../lib/utils";

export interface ChipProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "onChange"> {
  /**
   * 토글 선택 상태. 명시하면 aria-pressed 로 알린다(필터·정렬 토글).
   * 누르면 이동·실행만 하는 칩(추천 질문·섹션 이동)은 생략한다 — 그때는 aria-pressed 가 붙지 않는다.
   */
  selected?: boolean;
  /** 토글 콜백. 주면 클릭 시 !selected 로 호출된다. */
  onSelectedChange?: (selected: boolean) => void;
  leading?: React.ReactNode;
  /** 토글이 아닌 칩을 선택 모양(반전)으로만 그린다 — 적용 중 필터 해제 칩처럼 aria-pressed 가 맞지 않는 경우. */
  emphasized?: boolean;
}

/**
 * Chip — 필터·태그 선택용 토글 버튼(selected 를 줄 때만 aria-pressed), 또는 동작 칩.
 * 여러 개를 가로 스크롤 줄에 둘 때는 부모에 `flex gap-2 overflow-x-auto` 를 쓴다.
 */
function chipClass(selected: boolean, className?: string) {
  return cn(
    // 시각 32px + 위아래 6px 확장 = 44px 터치 영역
    "relative inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full border px-3 text-sm",
    "before:absolute before:inset-x-0 before:-inset-y-1.5 before:content-['']",
    "transition-colors duration-fast disabled:pointer-events-none disabled:opacity-50",
    selected
      ? "border-transparent bg-inverse font-medium text-fg-inverse"
      : "border-line bg-canvas-elevated text-fg-secondary shadow-xs hover:bg-canvas-hover hover:text-fg",
    focusRing,
    className
  );
}

export interface ChipLinkProps extends Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, "href"> {
  href: string;
  selected?: boolean;
  /** 링크 부품(앱 Link). 없으면 <a> */
  linkAs?: React.ElementType;
  leading?: React.ReactNode;
}

/** 화면을 바꾸는 칩(기록 칩 등) — 진짜 링크라 ⌘/Ctrl·가운데 클릭으로 새 탭에 열린다. 선택은 aria-current. */
export function ChipLink({ href, selected = false, linkAs, leading, className, children, ...props }: ChipLinkProps) {
  const Tag: React.ElementType = linkAs ?? "a";
  return (
    <Tag href={href} aria-current={selected ? "page" : undefined} className={chipClass(selected, className)} {...props}>
      {leading ? <span className="-ml-0.5 inline-flex [&_svg]:h-4 [&_svg]:w-4">{leading}</span> : null}
      {children}
    </Tag>
  );
}

export const Chip = React.forwardRef<HTMLButtonElement, ChipProps>(function Chip(
  { selected, emphasized = false, onSelectedChange, onClick, leading, className, children, type = "button", ...props },
  ref
) {
  function handleClick(event: React.MouseEvent<HTMLButtonElement>) {
    onClick?.(event);
    if (!event.defaultPrevented) onSelectedChange?.(!selected);
  }

  return (
    <button
      ref={ref}
      type={type}
      aria-pressed={selected}
      onClick={handleClick}
      className={chipClass(Boolean(selected) || emphasized, className)}
      {...props}
    >
      {leading ? <span className="-ml-0.5 inline-flex [&_svg]:h-4 [&_svg]:w-4">{leading}</span> : null}
      {children}
    </button>
  );
});
