import { ErrorState, Field, ListSkeleton, Select, Spinner, Switch } from "@/src/shared/ui/kit";
import { READER_LEVEL_LABELS_BY_LOCALE, describeSettingsError, useLearningSettings, useSaveLearningSettings } from "@/src/entities/punisher";
import {
  READER_LEVELS,
  describePunisherError,
  type ChosenReaderLevel,
  type ExampleLanguage,
  type LabOS,
  type LearningSettingsPatch,
} from "@/src/shared/api/punisher";
import { ReviewSettingsFields } from "./ReviewSettingsFields";
import type { SettingsField } from "./reviewSettings";
import { DEFAULT_LOCALE, LOCALES, isLocale, useLocale, useT } from "@/src/shared/i18n";
import { LANGUAGE_LABEL, LAB_OS_LABELS, LOCALE_NAMES, messages } from "./i18n";

const DAILY_MINUTE_OPTIONS = [15, 30, 45, 60] as const;
const LAB_OS_OPTIONS: readonly LabOS[] = ["macos", "linux", "none"];

function isReaderLevel(value: string): value is ChosenReaderLevel {
  return (READER_LEVELS as readonly string[]).includes(value);
}

/**
 * 학습 설정 — 언어 · 설명 수준 · 예시 코드 언어 · 하루 학습량 · 숙달 기준 · 복습 간격 · 하루 복습 시간 · 복습 채점 방식 · 예측 질문.
 * 바꾸면 그 필드만 바로 저장된다(PUT /me/settings 부분 업데이트). 예시 언어는 이후 생성되는 강의·질문 답변·
 * "이 부분 설명"에 적용된다(이미 만든 강의는 다시 만들지 않는다). 저장 오류는 그 필드 옆에(400 은 code 별 안내).
 */
export function LearningSettingsForm() {
  const settings = useLearningSettings();
  const save = useSaveLearningSettings();
  const t = useT(messages);
  const labOsLabels = useT(LAB_OS_LABELS);
  const readerLevelLabels = READER_LEVEL_LABELS_BY_LOCALE[useLocale()];

  if (settings.isLoading) { return <ListSkeleton rows={6} />; }
  if (settings.error || !settings.data) {
    return (
      <ErrorState
        compact
        message={describePunisherError(settings.error)}
        onRetry={() => settings.refetch()}
        retrying={settings.isRefetching}
      />
    );
  }

  const pending = save.isPending ? save.variables : undefined;
  const shown = { ...settings.data.settings, ...pending };
  const current = shown.exampleLanguage;
  const daily = shown.dailyMinutes ?? 30;
  const saveField = (patch: LearningSettingsPatch) => save.mutate(patch);
  const fieldError = (field: SettingsField): string | undefined => {
    if (!save.error || save.variables?.[field] === undefined) { return undefined; }
    return describeSettingsError(save.error);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-4">
        <Field label={LANGUAGE_LABEL} error={fieldError("locale")}>
          <Select
            value={shown.locale ?? DEFAULT_LOCALE}
            disabled={save.isPending}
            onChange={(event) => { if (isLocale(event.target.value)) { saveField({ locale: event.target.value }); } }}
          >
            {LOCALES.map((l) => (
              <option key={l} value={l}>
                {LOCALE_NAMES[l]}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t.readerLevel} hint={t.readerLevelHint} error={fieldError("readerLevel")}>
          <Select
            value={shown.readerLevel ?? ""}
            disabled={save.isPending}
            onChange={(event) => { if (isReaderLevel(event.target.value)) { saveField({ readerLevel: event.target.value }); } }}
          >
            {!shown.readerLevel && (
              <option value="" disabled>
                {t.readerLevelUnset}
              </option>
            )}
            {READER_LEVELS.map((level) => (
              <option key={level} value={level}>
                {readerLevelLabels[level]}
              </option>
            ))}
          </Select>
        </Field>
        <Field
          label={t.exampleLanguage}
          hint={t.exampleLanguageHint}
          error={fieldError("exampleLanguage")}
        >
          <Select value={current} disabled={save.isPending} onChange={(event) => saveField({ exampleLanguage: event.target.value as ExampleLanguage })}>
            {settings.data.exampleLanguages.map((lang) => (
              <option key={lang.id} value={lang.id}>
                {lang.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field
          label={t.dailyMinutes}
          hint={t.dailyMinutesHint}
          error={fieldError("dailyMinutes")}
        >
          <Select value={String(daily)} disabled={save.isPending} onChange={(event) => saveField({ dailyMinutes: Number(event.target.value) })}>
            {DAILY_MINUTE_OPTIONS.map((m) => (
              <option key={m} value={m}>
                {t.dailyOption(m)}
              </option>
            ))}
          </Select>
        </Field>
        <ReviewSettingsFields settings={shown} saving={save.isPending} onSave={saveField} fieldError={fieldError} />
        <Field label={t.labOs} error={fieldError("labOs")}>
          <Select value={shown.labOs ?? "macos"} disabled={save.isPending} onChange={(event) => saveField({ labOs: event.target.value as LabOS })}>
            {LAB_OS_OPTIONS.map((os) => (
              <option key={os} value={os}>
                {labOsLabels[os]}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t.pretest} error={fieldError("pretestEnabled")}>
          <Switch
            checked={shown.pretestEnabled ?? true}
            disabled={save.isPending}
            onCheckedChange={(checked) => saveField({ pretestEnabled: checked })}
          />
        </Field>
      </div>
      <p aria-live="polite" className="flex min-h-5 items-center gap-1.5 text-xs text-fg-muted">
        {save.isPending && <Spinner size="sm" label={null} />}
        {save.isPending && t.saving}
        {!save.isPending && save.isSuccess && t.saved}
      </p>
    </div>
  );
}
