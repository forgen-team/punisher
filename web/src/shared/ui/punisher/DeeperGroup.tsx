
import { useId, useState, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import { cn, focusRing } from "@/src/shared/ui/kit";

interface DeeperGroupProps {
  /** 원 강의 행(ListRow) */
  parent: ReactNode;
  /** "더 깊게" 본 강의 행들 — 접혀 있다가 펼치면 원 강의 아래 들여쓰기로 보인다 */
  children: ReactNode[];
}

/** 원 강의 행 + "더 깊게 본 강의 n개" 펼침. 심화 강의가 없으면 원 강의 행만. List 안에서 쓴다. */
export function DeeperGroup({ parent, children }: DeeperGroupProps) {
  const [open, setOpen] = useState(false);
  const listId = useId();
  if (children.length === 0) { return <>{parent}</>; }
  return (
    <>
      {parent}
      <li className="-mt-2 border-b border-line pb-1 last:border-b-0">
        <button
          type="button"
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => setOpen((prev) => !prev)}
          className={cn(
            "flex min-h-touch items-center gap-1 rounded-md pl-8 pr-2 text-xs font-medium text-fg-secondary hover:text-fg md:min-h-8",
            focusRing,
          )}
        >
          <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-90")} aria-hidden="true" />
          더 깊게 본 강의 {children.length}개
        </button>
        {open && (
          <ul id={listId} role="list" className="ml-8 border-l border-line pl-3">
            {children}
          </ul>
        )}
      </li>
    </>
  );
}
