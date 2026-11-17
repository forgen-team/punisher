
import { useId, useState, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import { cn, focusRing } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { blockMessages } from "../../i18n";

/** 접힌 내용(풀이·출력) — 버튼에 aria-expanded, 펼치면 아래에 내용을 그린다. */
export function Reveal({ label, children }: { label: string; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const id = useId();
  return (
    <div className="mt-2">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((prev) => !prev)}
        className={cn("inline-flex min-h-touch items-center gap-1 rounded-sm text-sm font-medium text-accent-text md:min-h-9", focusRing)}
      >
        <ChevronRight className={cn("size-4 transition-transform", open && "rotate-90")} aria-hidden="true" />
        {label}
      </button>
      <div id={id} hidden={!open}>
        {open && children}
      </div>
    </div>
  );
}

/** 구조 블록 데이터가 깨졌을 때의 작은 표시 — 화면의 나머지는 그대로 그린다. */
export function BlockError() {
  const t = useT(blockMessages);
  return (
    <p role="note" className="my-3 text-xs text-fg-muted">
      {t.broken}
    </p>
  );
}
