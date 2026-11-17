
import * as React from "react";
import { cn } from "../lib/utils";
import { useT } from "@/src/shared/i18n";
import { kitMessages } from "../lib/messages";
import { IconButton } from "./button";

export interface MenuItem {
  key: string;
  label: React.ReactNode;
  onSelect: () => void;
  icon?: React.ReactNode;
  /** 파괴적 동작(삭제 등) — danger 색 */
  danger?: boolean;
  disabled?: boolean;
}

export interface MenuProps {
  items: ReadonlyArray<MenuItem>;
  /** 트리거 버튼 aria-label (기본 "더 보기") */
  label?: string;
  /** 메뉴 정렬 기준 (기본 오른쪽) */
  align?: "start" | "end";
  /** 트리거 아이콘 교체 (기본 ⋯) */
  icon?: React.ReactNode;
  className?: string;
}

function MoreIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <circle cx="5" cy="12" r="1.75" />
      <circle cx="12" cy="12" r="1.75" />
      <circle cx="19" cy="12" r="1.75" />
    </svg>
  );
}

/**
 * Menu — 오버플로("⋯") 메뉴. role=menu/menuitem, ↑↓/Home/End, Esc 닫기 후 트리거로 포커스 복귀.
 * 행 안의 보조 동작은 여기에 모은다(행당 노출 버튼 최소화).
 */
export function Menu({ items, label, align = "end", icon, className }: MenuProps) {
  const t = useT(kitMessages);
  const triggerLabel = label ?? t.more;
  const [open, setOpen] = React.useState(false);
  const rootRef = React.useRef<HTMLDivElement>(null);
  const triggerRef = React.useRef<HTMLButtonElement>(null);
  const itemRefs = React.useRef<Array<HTMLButtonElement | null>>([]);
  const menuId = React.useId();

  const enabledIndexes = React.useMemo(
    () => items.flatMap((item, i) => (item.disabled ? [] : [i])),
    [items]
  );

  const close = React.useCallback((restoreFocus: boolean) => {
    setOpen(false);
    if (restoreFocus) triggerRef.current?.focus();
  }, []);

  // 열릴 때 한 번만 첫 항목으로 포커스 (items 참조가 바뀌어도 다시 튀지 않게 open만 의존)
  const firstEnabled = enabledIndexes[0] ?? 0;
  const firstEnabledRef = React.useRef(firstEnabled);
  firstEnabledRef.current = firstEnabled;
  React.useEffect(() => {
    if (!open) return;
    itemRefs.current[firstEnabledRef.current]?.focus();
    function handlePointerDown(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) close(false);
    }
    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [open, close]);

  function move(current: number, step: 1 | -1 | "first" | "last") {
    if (enabledIndexes.length === 0) return;
    let target: number | undefined;
    if (step === "first") target = enabledIndexes[0];
    else if (step === "last") target = enabledIndexes[enabledIndexes.length - 1];
    else {
      const pos = enabledIndexes.indexOf(current);
      const next = (pos + step + enabledIndexes.length) % enabledIndexes.length;
      target = enabledIndexes[next];
    }
    if (target !== undefined) itemRefs.current[target]?.focus();
  }

  function handleMenuKeyDown(event: React.KeyboardEvent, index: number) {
    if (event.key === "ArrowDown") move(index, 1);
    else if (event.key === "ArrowUp") move(index, -1);
    else if (event.key === "Home") move(index, "first");
    else if (event.key === "End") move(index, "last");
    else if (event.key === "Escape") close(true);
    else if (event.key === "Tab") close(false);
    else return;
    if (event.key !== "Tab") event.preventDefault();
  }

  function handleTriggerKeyDown(event: React.KeyboardEvent) {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      setOpen(true);
    }
  }

  return (
    <div ref={rootRef} className={cn("relative inline-flex", className)}>
      <IconButton
        ref={triggerRef}
        aria-label={triggerLabel}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((prev) => !prev)}
        onKeyDown={handleTriggerKeyDown}
      >
        {icon ?? <MoreIcon />}
      </IconButton>
      {open ? (
        <div
          id={menuId}
          role="menu"
          aria-label={triggerLabel}
          className={cn(
            "absolute top-full z-panel mt-1 min-w-[180px] overflow-hidden rounded-lg border border-line p-1",
            "bg-canvas-elevated shadow-overlay animate-fade-in motion-reduce:animate-none",
            align === "end" ? "right-0" : "left-0"
          )}
        >
          {items.map((item, index) => (
            <button
              key={item.key}
              ref={(node) => {
                itemRefs.current[index] = node;
              }}
              type="button"
              role="menuitem"
              tabIndex={-1}
              disabled={item.disabled}
              onKeyDown={(event) => handleMenuKeyDown(event, index)}
              onClick={() => {
                close(true);
                item.onSelect();
              }}
              className={cn(
                "flex min-h-touch w-full items-center gap-2 rounded-md px-2.5 text-left text-sm outline-none md:min-h-8",
                "hover:bg-canvas-hover focus-visible:bg-canvas-hover disabled:opacity-50",
                "[&_svg]:h-4 [&_svg]:w-4",
                item.danger ? "text-danger" : "text-fg"
              )}
            >
              {item.icon ? <span className="inline-flex text-fg-muted">{item.icon}</span> : null}
              {item.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
