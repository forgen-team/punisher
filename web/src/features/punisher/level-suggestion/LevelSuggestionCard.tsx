import { Button } from "@forgen-team/ui";
import { READER_LEVEL_LABELS_BY_LOCALE, describeSettingsError, useDismissLevelSuggestion, useLearningSettings, useSaveLearningSettings } from "@/src/entities/punisher";
import { levelSuggestionOf } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 수준 보정 카드(④) — 최근 '쉽게'를 자주 눌러 서버가 한 단계 쉬운 수준을 제안하면 강의 화면 위에 한 줄로 띄운다.
 * 바꾸기 = 설정 PUT readerLevel, 괜찮아요 = 7일 숨김.
 */
export function LevelSuggestionCard() {
  const t = useT(messages);
  const locale = useLocale();
  const settings = useLearningSettings();
  const save = useSaveLearningSettings();
  const dismiss = useDismissLevelSuggestion();
  const suggestion = levelSuggestionOf(settings.data);
  if (!suggestion || save.isSuccess || dismiss.isSuccess) { return null; }
  const error = save.error ?? dismiss.error;
  const busy = save.isPending || dismiss.isPending;
  return (
    <section aria-label={t.label} className="mb-3 flex flex-wrap items-center gap-2 rounded-lg border border-line bg-canvas-elevated px-3 py-2 text-sm shadow-xs">
      <div className="min-w-0 flex-1">
        <p className="font-medium text-fg">{t.title}</p>
        <p className="text-xs text-fg-secondary">{t.target(READER_LEVEL_LABELS_BY_LOCALE[locale][suggestion.to])}</p>
      </div>
      <div className="flex items-center gap-1">
        <Button size="sm" variant="secondary" loading={save.isPending} disabled={busy} onClick={() => save.mutate({ readerLevel: suggestion.to })}>
          {t.change}
        </Button>
        <Button size="sm" variant="ghost" loading={dismiss.isPending} disabled={busy} onClick={() => dismiss.mutate()}>
          {t.dismiss}
        </Button>
      </div>
      {error ? <p role="alert" className="w-full text-xs text-danger">{describeSettingsError(error)}</p> : null}
    </section>
  );
}
