import { cn, focusRing } from "@/src/shared/lib/cn";
import type { Confidence } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { CONFIDENCE_LABELS, CONFIDENCE_ORDER, messages } from "../i18n";

/** 답 전에 고르는 확신도(선택) — 다시 누르면 해제. 판정은 바꾸지 않는다(v3 §3). */
export function ConfidencePicker({ value, onChange }: { value?: Confidence; onChange: (v?: Confidence) => void }) {
  const locale = useLocale();
  const t = useT(messages);
  return (
    <div role="radiogroup" aria-label={t.confidence} className="flex gap-1.5">
      {CONFIDENCE_ORDER.map((c) => {
        const on = value === c;
        return (
          <button
            key={c}
            type="button"
            role="radio"
            aria-checked={on}
            onClick={() => onChange(on ? undefined : c)}
            className={cn(
              "min-h-9 rounded-full border px-3 text-sm transition-colors",
              on ? "border-accent bg-accent-subtle font-semibold text-accent-text" : "border-line text-fg-secondary hover:bg-canvas-subtle",
              focusRing,
            )}
          >
            {CONFIDENCE_LABELS[locale][c]}
          </button>
        );
      })}
    </div>
  );
}
