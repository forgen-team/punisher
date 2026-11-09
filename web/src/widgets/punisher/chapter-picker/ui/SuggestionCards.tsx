
import { Check } from "lucide-react";
import { cn, focusRing } from "@/src/shared/ui/kit";
import type { ChapterSuggestion } from "@/src/shared/api/punisher";
import { MAX_PICKS, SUGGESTION_KIND_LABELS } from "./constants";

interface SuggestionCardsProps {
  suggestions: ChapterSuggestion[];
  selected: ReadonlySet<string>;
  onToggle: (id: string) => void;
}

/** 자동 제안 카드(여러 개 고를 수 있음) — 카드마다 왜 추천하는지 한 줄. */
export function SuggestionCards({ suggestions, selected, onToggle }: SuggestionCardsProps) {
  const full = selected.size >= MAX_PICKS;
  return (
    <ul role="list" className="grid gap-2 fold:grid-cols-2">
      {suggestions.map((s) => {
        const on = selected.has(s.id);
        return (
          <li key={s.id}>
            <button
              type="button"
              aria-pressed={on}
              disabled={!on && full}
              onClick={() => onToggle(s.id)}
              className={cn(
                "flex h-full w-full items-start gap-3 rounded-lg border p-3 text-left transition-colors disabled:opacity-50",
                on ? "border-accent-solid bg-accent-subtle" : "border-line bg-canvas-elevated hover:bg-canvas-hover",
                focusRing,
              )}
            >
              <span
                aria-hidden="true"
                className={cn(
                  "mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-md border",
                  on ? "border-accent-solid bg-accent-solid text-accent-on" : "border-line-strong",
                )}
              >
                {on && <Check className="size-3.5" />}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-xs font-medium text-fg-muted">{SUGGESTION_KIND_LABELS[s.kind] ?? s.kind}</span>
                <span className="block text-base font-semibold text-fg">{s.title}</span>
                <span className="mt-0.5 block text-sm text-fg-secondary">{s.reason}</span>
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
