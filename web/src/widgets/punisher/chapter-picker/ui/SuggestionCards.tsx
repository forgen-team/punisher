
import { Check } from "lucide-react";
import { EmptyState, Skeleton, cn, focusRing } from "@/src/shared/ui/kit";
import type { ChapterSuggestion } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { MAX_PICKS } from "./constants";
import { SUGGESTION_KIND_LABELS, messages } from "../i18n";

interface SuggestionCardsProps {
  suggestions: ChapterSuggestion[];
  selected: ReadonlySet<string>;
  onToggle: (id: string) => void;
}

/** 자동 제안 카드(여러 개 고를 수 있음) — 카드마다 왜 추천하는지 한 줄. */
export function SuggestionCards({ suggestions, selected, onToggle }: SuggestionCardsProps) {
  const t = useT(messages);
  const kindLabels = useT(SUGGESTION_KIND_LABELS);
  const full = selected.size >= MAX_PICKS;
  if (suggestions.length === 0) {
    return <EmptyState compact title={t.suggestionsEmptyTitle} description={t.suggestionsEmptyDescription} />;
  }
  return (
    <ul role="list" className="grid gap-2 fold:grid-cols-2">
      {suggestions.map((s) => {
        const on = selected.has(s.id);
        return (
          <li key={s.id}>
            <button
              type="button"
              role="checkbox"
              aria-checked={on}
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
                  // 여러 개 고르는 선택이라 체크박스 모양(각진 네모) — 동그라미면 하나만 고르는 라디오로 읽힌다.
                  "mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-sm border-2",
                  on ? "border-accent-solid bg-accent-solid text-accent-on" : "border-line-strong",
                )}
              >
                {on && <Check className="size-3.5" />}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-xs font-medium text-fg-muted">{kindLabels[s.kind] ?? s.kind}</span>
                <span className="block text-base font-semibold text-fg">{s.title}</span>
                <span className="mt-0.5 block break-keep text-sm text-fg-secondary">{s.reason}</span>
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}

/** 불러오는 동안 카드와 같은 격자·높이를 잡아 둬서 로드 뒤 화면이 밀리지 않게 한다. */
export function SuggestionCardsSkeleton() {
  return (
    <ul role="list" aria-hidden="true" className="grid gap-2 fold:grid-cols-2">
      {Array.from({ length: 4 }, (_, index) => (
        <li key={index} className="flex items-start gap-3 rounded-lg border border-line bg-canvas-elevated p-3">
          <Skeleton variant="block" className="mt-0.5 size-5 shrink-0 rounded-sm" />
          <Skeleton lines={3} className="flex-1" />
        </li>
      ))}
    </ul>
  );
}
