
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
}

/**
 * SegmentedTabs — role="tablist", 화살표/Home/End 키, roving tabindex.
 * 회색 트랙 위에 선택된 탭만 떠 있는 알약(배경·그림자)으로 표시한다.
 */
export function SegmentedTabs<V extends string>({
  items, value, onChange, fill = false, className, "aria-label": ariaLabel,
}: SegmentedTabsProps<V>) {
  const refs = React.useRef<Array<HTMLButtonElement | null>>([]);

  function focusAt(start: number, step: 1 | -1) {
    const count = items.length;
    for (let i = 1; i <= count; i++) {
      const index = (start + step * i + count) % count;
      const item = items[index];
      if (item && !item.disabled) {
        refs.current[index]?.focus();
        onChange(item.value);
        return;
      }
    }
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLButtonElement>, index: number) {
    if (event.key === "ArrowRight") focusAt(index, 1);
    else if (event.key === "ArrowLeft") focusAt(index, -1);
    else if (event.key === "Home") focusAt(-1, 1);
    else if (event.key === "End") focusAt(items.length, -1);
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
        return (
          <button
            key={item.value}
            ref={(node) => {
              refs.current[index] = node;
            }}
            type="button"
            role="tab"
            aria-selected={selected}
            aria-controls={item.panelId}
            tabIndex={selected ? 0 : -1}
            disabled={item.disabled}
            onClick={() => onChange(item.value)}
            onKeyDown={(event) => handleKeyDown(event, index)}
            className={cn(
              // 시각 36px + 트랙 여백 = 터치 40px 이상, 위아래 확장으로 44px
              "relative inline-flex h-9 shrink-0 items-center justify-center whitespace-nowrap rounded-md px-3 text-sm tracking-ui",
              "before:absolute before:inset-x-0 before:-inset-y-1 before:content-['']",
              "transition-[color,background-color,box-shadow] duration-fast disabled:opacity-50",
              fill && "flex-1",
              selected
                ? "bg-canvas-elevated font-medium text-fg shadow-sm"
                : "text-fg-muted hover:text-fg",
              focusRing,
              "focus-visible:ring-offset-0 focus-visible:ring-inset"
            )}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
