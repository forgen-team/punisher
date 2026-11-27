
import * as React from "react";
import { cn, focusRing } from "../lib/utils";

export interface SegmentedTabItem<V extends string> {
  value: V;
  label: React.ReactNode;
  /** 탭 패널 id (aria-controls) */
  panelId?: string;
  disabled?: boolean;
}

export interface SegmentedTabsProps<V extends string> {
  items: ReadonlyArray<SegmentedTabItem<V>>;
  /** 제어형: URL 검색 파라미터 등에서 읽은 값 */
  value: V;
  /** 선택 변경 — 보통 router.replace(?tab=...) 를 호출 */
  onChange: (value: V) => void;
  /** 탭 목록 이름 (예: "RSS 보기") */
  "aria-label": string;
  /** 꽉 찬 너비로 균등 분할 (모바일 상단 세그먼트) */
  fill?: boolean;
  className?: string;
  /**
   * 탭이 화면 이동이면 링크로 그린다(⌘/Ctrl·가운데 클릭으로 새 탭). 값마다 주소를 주고, 링크 부품(앱 Link)을 넘긴다.
   * 보통 클릭은 onChange 로 오므로 이동은 부르는 쪽이 정한다(지금 탭을 다시 누르면 처음 화면 등).
   */
  hrefOf?: (value: V) => string;
  linkAs?: React.ElementType;
}

/**
 * SegmentedTabs — role="tablist", roving tabindex. 화살표/Home/End 는 포커스만 옮기고 Enter/Space 로 고른다
 * (Witems 화면 배치 규칙 §2 — 화살표만으로 화면이 바뀌거나 이동하지 않게).
 * 회색 트랙 위에 선택된 탭만 떠 있는 알약(배경·그림자)으로 표시한다.
 */
export function SegmentedTabs<V extends string>({
  items, value, onChange, fill = false, className, "aria-label": ariaLabel, hrefOf, linkAs,
}: SegmentedTabsProps<V>) {
  const refs = React.useRef<Array<HTMLElement | null>>([]);
  const selectedIndex = Math.max(0, items.findIndex((item) => item.value === value));
  // 화살표로 옮긴 포커스 자리(roving tabindex) — 고르기 전까지 선택은 그대로다.
  const [focusIndex, setFocusIndex] = React.useState(selectedIndex);
  React.useEffect(() => setFocusIndex(selectedIndex), [selectedIndex]);

  function focusAt(start: number, step: 1 | -1) {
    const count = items.length;
    for (let i = 1; i <= count; i++) {
      const index = (start + step * i + count) % count;
      const item = items[index];
      if (item && !item.disabled) {
        setFocusIndex(index);
        refs.current[index]?.focus();
        return;
      }
    }
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLElement>, index: number) {
    if (event.key === "ArrowRight") focusAt(index, 1);
    else if (event.key === "ArrowLeft") focusAt(index, -1);
    else if (event.key === "Home") focusAt(-1, 1);
    else if (event.key === "End") focusAt(items.length, -1);
    // 링크는 Space 로 눌리지 않으므로 직접 누른다(버튼은 Enter/Space 가 기본으로 click).
    else if (event.key === " " && hrefOf) event.currentTarget.click();
    else return;
    event.preventDefault();
  }

  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      className={cn(
        "flex gap-0.5 overflow-x-auto rounded-lg border border-line-subtle bg-canvas-subtle p-0.5 [scrollbar-width:none]",
        fill ? "w-full" : "w-fit max-w-full",
        className
      )}
    >
      {items.map((item, index) => {
        const selected = item.value === value;
        const Tag: React.ElementType = hrefOf ? (linkAs ?? "a") : "button";
        const own = hrefOf
          ? {
              href: hrefOf(item.value),
              "aria-disabled": item.disabled || undefined,
              onClick: (event: React.MouseEvent<HTMLAnchorElement>) => {
                // 새 탭(⌘/Ctrl·가운데·Shift)은 브라우저에 맡긴다.
                if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
                event.preventDefault();
                if (!item.disabled) onChange(item.value);
              },
            }
          : { type: "button", disabled: item.disabled, onClick: () => onChange(item.value) };
        return (
          <Tag
            key={item.value}
            ref={(node: HTMLElement | null) => {
              refs.current[index] = node;
            }}
            role="tab"
            aria-selected={selected}
            aria-controls={item.panelId}
            tabIndex={index === focusIndex ? 0 : -1}
            onKeyDown={(event: React.KeyboardEvent<HTMLElement>) => handleKeyDown(event, index)}
            {...own}
            className={cn(
              // 시각 36px + 트랙 여백 = 터치 40px 이상, 위아래 확장으로 44px
              "relative inline-flex h-9 shrink-0 items-center justify-center whitespace-nowrap rounded-md px-3 text-sm tracking-ui",
              "before:absolute before:inset-x-0 before:-inset-y-1 before:content-['']",
              "transition-[color,background-color,box-shadow] duration-fast disabled:opacity-50 aria-disabled:opacity-50",
              fill && "flex-1",
              selected
                ? "bg-canvas-selected font-medium text-fg shadow-sm"
                : "text-fg-muted hover:text-fg",
              focusRing,
              "focus-visible:ring-offset-0 focus-visible:ring-inset"
            )}
          >
            {item.label}
          </Tag>
        );
      })}
    </div>
  );
}
