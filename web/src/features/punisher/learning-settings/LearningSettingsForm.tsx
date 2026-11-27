import { ErrorState, ListSkeleton } from "@forgen-team/ui";
import { READER_LEVEL_LABELS_BY_LOCALE, describeSettingsError, useLearningSettings, useSaveLearningSettings } from "@/src/entities/punisher";
import {
  READER_LEVELS,
  describePunisherError,
  type ChosenReaderLevel,
  type ExampleLanguage,
  type LabOS,
  type LearningSettingsPatch,
} from "@/src/shared/api/punisher";
import { DEFAULT_LOCALE, LOCALES, isLocale, useLocale, useT } from "@/src/shared/i18n";
import { LANGUAGE_LABEL, LAB_OS_LABELS, LOCALE_NAMES, messages } from "./i18n";
import { SelectRow, SwitchRow } from "./SettingRow";
import { useFieldSaves } from "./useFieldSaves";

const DAILY_MINUTE_OPTIONS = [15, 30, 45, 60] as const;
const LAB_OS_OPTIONS: readonly LabOS[] = ["macos", "linux", "none"];

function isReaderLevel(value: string): value is ChosenReaderLevel {
  return (READER_LEVELS as readonly string[]).includes(value);
}

/**
 * 설정 › 학습 구역 — 언어 · 설명 수준 · 예시 코드 언어 · 하루 학습량 · 실습 환경 · 강의 전 예측 질문.
 * 고르거나 켜면 그 필드만 바로 저장된다(PUT /me/settings 부분 업데이트, 그 자리에 "저장했어요"). 예시 언어는 이후
 * 생성되는 강의·질문 답변·"이 부분 설명"에 적용된다(이미 만든 강의는 다시 만들지 않는다). 저장 오류는 그 칸 아래에.
 * 저장하는 동안에도 다른 칸을 계속 바꿀 수 있다(점검 18) — 아직 저장 중인 값은 모두 화면에 먼저 보인다.
 * (복습 항목은 ReviewSettingsFields — 같은 설정 PUT 을 쓰지만 구역이 따로다.)
 */
export function LearningSettingsForm() {
  const settings = useLearningSettings();
  const save = useSaveLearningSettings();
  const t = useT(messages);
  const labOsLabels = useT(LAB_OS_LABELS);
  const readerLevelLabels = READER_LEVEL_LABELS_BY_LOCALE[useLocale()];
  const { unsaved, errors, savedKey, saveField } = useFieldSaves<LearningSettingsPatch>(save.mutateAsync, describeSettingsError);

  if (settings.isLoading) { return <ListSkeleton rows={6} />; }
  if (settings.error || !settings.data) {
    return <ErrorState title={describePunisherError(settings.error)} onRetry={() => { void settings.refetch(); }} />;
  }

  const shown = { ...settings.data.settings, ...unsaved };
  const row = (field: keyof LearningSettingsPatch) => ({ saved: savedKey === field, error: errors[field] });
  return (
    <div className="flex flex-col">
      <SelectRow
        label={LANGUAGE_LABEL}
        {...row("locale")}
        value={shown.locale ?? DEFAULT_LOCALE}
        onChange={(v) => { if (isLocale(v)) { saveField({ locale: v }); } }}
        options={LOCALES.map((l) => ({ value: l, label: LOCALE_NAMES[l] }))}
      />
      <SelectRow
        label={t.readerLevel}
        hint={t.readerLevelHint}
        {...row("readerLevel")}
        value={shown.readerLevel ?? ""}
        placeholder={t.readerLevelUnset}
        onChange={(v) => { if (isReaderLevel(v)) { saveField({ readerLevel: v }); } }}
        options={READER_LEVELS.map((level) => ({ value: level, label: readerLevelLabels[level] }))}
      />
      <SelectRow
        label={t.exampleLanguage}
        hint={t.exampleLanguageHint}
        {...row("exampleLanguage")}
        value={shown.exampleLanguage}
        onChange={(v) => saveField({ exampleLanguage: v as ExampleLanguage })}
        options={settings.data.exampleLanguages.map((lang) => ({ value: lang.id, label: lang.label }))}
      />
      <SelectRow
        label={t.dailyMinutes}
        hint={t.dailyMinutesHint}
        {...row("dailyMinutes")}
        value={String(shown.dailyMinutes ?? 30)}
        onChange={(v) => saveField({ dailyMinutes: Number(v) })}
        options={DAILY_MINUTE_OPTIONS.map((m) => ({ value: String(m), label: t.dailyOption(m) }))}
      />
      <SelectRow
        label={t.labOs}
        {...row("labOs")}
        value={shown.labOs ?? "macos"}
        onChange={(v) => saveField({ labOs: v as LabOS })}
        options={LAB_OS_OPTIONS.map((os) => ({ value: os, label: labOsLabels[os] }))}
      />
      <SwitchRow
        label={t.pretest}
        {...row("pretestEnabled")}
        checked={shown.pretestEnabled ?? true}
        onChange={(checked) => saveField({ pretestEnabled: checked })}
      />
    </div>
  );
}
