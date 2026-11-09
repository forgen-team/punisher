import * as React from "react";
import { cn, focusRing } from "../lib/utils";

export interface ChipProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "onChange"> {
  /** 선택 상태. 강조색은 선택에만 쓰인다. */
  selected?: boolean;
  /** 토글 콜백. 주면 클릭 시 !selected 로 호출된다. */
  onSelectedChange?: (selected: boolean) => void;
  leading?: React.ReactNode;
}

/**
 * Chip — 필터·태그 선택용 토글 버튼(aria-pressed).
 * 여러 개를 가로 스크롤 줄에 둘 때는 부모에 `flex gap-2 overflow-x-auto` 를 쓴다.
 */
export const Chip = React.forwardRef<HTMLButtonElement, ChipProps>(function Chip(
  { selected = false, onSelectedChange, onClick, leading, className, children, type = "button", ...props },
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
      className={cn(
        // 시각 32px + 위아래 6px 확장 = 44px 터치 영역
        "relative inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full border px-3 text-sm",
        "before:absolute before:inset-x-0 before:-inset-y-1.5 before:content-['']",
        "transition-colors duration-fast disabled:pointer-events-none disabled:opacity-50",
        selected
          ? "border-transparent bg-inverse font-medium text-fg-inverse"
          : "border-line bg-canvas-elevated text-fg-secondary shadow-xs hover:bg-canvas-hover hover:text-fg",
        focusRing,
        className
      )}
      {...props}
    >
      {leading ? <span className="-ml-0.5 inline-flex [&_svg]:h-4 [&_svg]:w-4">{leading}</span> : null}
      {children}
    </button>
  );
});
