import { useState } from "react";
import { Chip, ErrorState, Field, ListSkeleton, Select, Switch } from "@/src/shared/ui/kit";
import { useSaveStudyPrefs, useStudyPrefs } from "@/src/entities/punisher";
import { describePunisherError, type MemoryModel, type StudyPrefsPatch } from "@/src/shared/api/punisher";
import { isWitems } from "@/src/shared/config/runtime";
import { setUiPrefs, useUiPrefs, type FontSize, type LineWidth } from "@/src/shared/lib/ui-prefs";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { studyMessages } from "./i18n";

const RETENTIONS = [80, 85, 90, 95] as const;
const HOURS = Array.from({ length: 24 }, (_, h) => h);
const WEEKDAYS = [0, 1, 2, 3, 4, 5, 6] as const;

/**
 * 설정 > 학습의 복습·기억(W5a): 간격 방식(FSRS/상자) · 목표 유지율 · 쉬는 요일 · 복습 알림(단독판 macOS 만).
 * 바꾸면 그 칸만 바로 저장한다(학습 설정과 같은 방식). 저장 중에도 다른 칸을 바꿀 수 있다.
 */
export function StudySettingsFields() {
  const t = useT(studyMessages);
  const prefs = useStudyPrefs();
  const save = useSaveStudyPrefs();
  const [unsaved, setUnsaved] = useState<StudyPrefsPatch>({});
  if (prefs.isLoading) { return <ListSkeleton rows={3} />; }
  if (prefs.error || !prefs.data) {
    return <ErrorState compact message={describePunisherError(prefs.error)} onRetry={() => prefs.refetch()} retrying={prefs.isRefetching} />;
  }
  const shown = { ...prefs.data, ...unsaved };
  const saveField = (patch: StudyPrefsPatch) => {
    setUnsaved((prev) => ({ ...prev, ...patch }));
    save.mutateAsync(patch)
      .catch((err: unknown) => console.warn("punisher: study prefs save failed", err))
      .finally(() => setUnsaved((prev) => {
        const next: Record<string, unknown> = { ...prev };
        for (const key of Object.keys(patch)) { delete next[key]; }
        return next as StudyPrefsPatch;
      }));
  };
  const toggleRest = (day: number) => {
    const set = new Set(shown.restDays);
    if (set.has(day)) { set.delete(day); } else if (set.size < 6) { set.add(day); }
    saveField({ restDays: [...set].sort((a, b) => a - b) });
  };

  return (
    <div className="flex flex-col gap-4">
      <InlineError error={save.error} />
      <Field label={t.memoryModel} hint={t.memoryModelHint}>
        <Select value={shown.memoryModel} onChange={(e) => saveField({ memoryModel: e.target.value as MemoryModel })}>
          <option value="fsrs">{t.modelFsrs}</option>
          <option value="ladder">{t.modelLadder}</option>
        </Select>
      </Field>
      {shown.memoryModel === "fsrs" && (
        <Field label={t.retention} hint={t.retentionHint}>
          <Select value={String(Math.round(shown.targetRetention * 100))} onChange={(e) => saveField({ targetRetention: Number(e.target.value) / 100 })}>
            {RETENTIONS.map((p) => <option key={p} value={p}>{t.retentionOption(p)}</option>)}
          </Select>
        </Field>
      )}
      <fieldset className="flex flex-col gap-1.5">
        <legend className="text-sm font-medium text-fg">{t.restDays}</legend>
        <div className="flex flex-wrap gap-1.5">
          {WEEKDAYS.map((day) => (
            <Chip key={day} selected={shown.restDays.includes(day)} onClick={() => toggleRest(day)}>{t.weekday(day)}</Chip>
          ))}
        </div>
        <p className="text-xs text-fg-muted">{t.restDaysHint}</p>
      </fieldset>
      {isWitems() ? (
        <p className="text-xs text-fg-muted">{t.reminderWitems}</p>
      ) : (
        <Field label={t.reminder} hint={t.reminderHint}>
          <Select value={String(shown.reminderHour)} onChange={(e) => saveField({ reminderHour: Number(e.target.value) })}>
            <option value="-1">{t.reminderOff}</option>
            {HOURS.map((h) => <option key={h} value={h}>{t.reminderAt(h)}</option>)}
          </Select>
        </Field>
      )}
    </div>
  );
}

/** 설정 > 읽기(이 기기만): 강의 글자 크기 · 줄 폭 · 복습 단축키. */
export function ReadingSettingsFields() {
  const t = useT(studyMessages);
  const ui = useUiPrefs();
  return (
    <div className="flex flex-col gap-4">
      <Field label={t.fontSize}>
        <Select value={ui.fontSize} onChange={(e) => setUiPrefs({ fontSize: e.target.value as FontSize })}>
          <option value="sm">{t.fontSm}</option>
          <option value="md">{t.fontMd}</option>
          <option value="lg">{t.fontLg}</option>
        </Select>
      </Field>
      <Field label={t.lineWidth}>
        <Select value={ui.lineWidth} onChange={(e) => setUiPrefs({ lineWidth: e.target.value as LineWidth })}>
          <option value="narrow">{t.widthNarrow}</option>
          <option value="normal">{t.widthNormal}</option>
          <option value="wide">{t.widthWide}</option>
        </Select>
      </Field>
      <Field label={t.shortcuts} hint={t.shortcutsHint}>
        <Switch checked={ui.shortcuts} onCheckedChange={(checked) => setUiPrefs({ shortcuts: checked })} />
      </Field>
      <p className="text-xs text-fg-muted">{t.deviceHint}</p>
    </div>
  );
}
