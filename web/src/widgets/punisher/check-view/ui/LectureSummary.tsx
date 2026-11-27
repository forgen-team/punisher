
import { useId, useState } from "react";
import { ChevronRight } from "lucide-react";
import { cn, focusRing } from "@/src/shared/lib/cn";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** '이 강의 한 장 요약'(v2.2 §6) — 기본은 접힘, 펼치면 강의의 @@summary 를 다시 보여 준다. 요약이 없으면 그리지 않는다. */
export function LectureSummary({ text }: { text?: string }) {
  const t = useT(messages);
  const [open, setOpen] = useState(false);
  const id = useId();
  if (!text) { return null; }
  return (
    <div className="flex flex-col">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((prev) => !prev)}
        className={cn("inline-flex min-h-touch items-center gap-1 self-start rounded-sm text-sm font-medium text-accent-text md:min-h-9", focusRing)}
      >
        <ChevronRight className={cn("size-4 transition-transform", open && "rotate-90")} aria-hidden="true" />
        {t.lectureSummary}
      </button>
      <div id={id} hidden={!open} className="rounded-md border border-line bg-canvas-elevated px-3 py-1 text-sm">
        {open && <Markdown>{text}</Markdown>}
      </div>
    </div>
  );
}
