
import * as React from "react";
import { cn } from "../lib/utils";
import { useT } from "@/src/shared/i18n";
import { kitMessages } from "../lib/messages";
import { eventWithin } from "@/src/shared/lib/dom-root";
import { IconButton } from "./button";

export interface MenuItem {
  key: string;
  label: React.ReactNode;
  /** 실행 항목. href 가 있으면 이동 항목(링크 — ⌘/Ctrl·가운데 클릭 새 탭) */
  onSelect?: () => void;
  href?: string;
  icon?: React.ReactNode;
  /** 파괴적 동작(삭제 등) — danger 색, 보통 맨 아래 구분선 뒤 */
  danger?: boolean;
  disabled?: boolean;
  /** 이 항목 앞에 구분선(묶음 나누기: 상태 변경 / 이동·설정 / 위험) */
  separatorBefore?: boolean;
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
  /** href 항목을 그릴 링크 부품(앱 Link). 없으면 <a> */
  linkAs?: React.ElementType;
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
export function Menu({ items, label, align = "end", icon, className, linkAs }: MenuProps) {
  const t = useT(kitMessages);
  const triggerLabel = label ?? t.more;
  const [open, setOpen] = React.useState(false);
  const rootRef = React.useRef<HTMLDivElement>(null);
  const triggerRef = React.useRef<HTMLButtonElement>(null);
  const itemRefs = React.useRef<Array<HTMLElement | null>>([]);
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
      // Witems(Shadow DOM)에서는 document 에서 본 target 이 호스트라 contains 로는 항상 바깥이 된다 — composedPath 로 판정.
      if (!eventWithin(event, rootRef.current)) close(false);
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
          {items.map((item, index) => {
            const itemClass = cn(
              "flex min-h-touch w-full items-center gap-2 rounded-md px-2.5 text-left text-sm outline-none md:min-h-8",
              "hover:bg-canvas-hover focus-visible:bg-canvas-hover disabled:opacity-50 aria-disabled:opacity-50",
              "[&_svg]:h-4 [&_svg]:w-4",
              item.danger ? "text-danger" : "text-fg"
            );
            const body = (
              <>
                {item.icon ? <span className="inline-flex text-fg-muted">{item.icon}</span> : null}
                {item.label}
              </>
            );
            const setRef = (node: HTMLElement | null) => {
              itemRefs.current[index] = node;
            };
            const LinkTag: React.ElementType = linkAs ?? "a";
            return (
              <React.Fragment key={item.key}>
                {item.separatorBefore && index > 0 ? <div role="separator" className="my-1 h-px bg-line" /> : null}
                {item.href && !item.disabled ? (
                  <LinkTag
                    ref={setRef}
                    href={item.href}
                    role="menuitem"
                    tabIndex={-1}
                    onKeyDown={(event: React.KeyboardEvent) => handleMenuKeyDown(event, index)}
                    onClick={() => {
                      close(false);
                      item.onSelect?.();
                    }}
                    className={itemClass}
                  >
                    {body}
                  </LinkTag>
                ) : (
                  <button
                    ref={setRef}
                    type="button"
                    role="menuitem"
                    tabIndex={-1}
                    disabled={item.disabled}
                    onKeyDown={(event) => handleMenuKeyDown(event, index)}
                    onClick={() => {
                      close(true);
                      item.onSelect?.();
                    }}
                    className={itemClass}
                  >
                    {body}
                  </button>
                )}
              </React.Fragment>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
