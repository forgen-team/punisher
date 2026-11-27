import { useState } from "react";
import { Chip, ErrorState, ListSkeleton, Select, Switch } from "@forgen-team/ui";
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
    return <ErrorState title={describePunisherError(prefs.error)} onRetry={() => { void prefs.refetch(); }} />;
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
      <Select
        label={t.memoryModel}
        description={t.memoryModelHint}
        value={shown.memoryModel}
        onValueChange={(v) => saveField({ memoryModel: v as MemoryModel })}
        options={[
          { value: "fsrs", label: t.modelFsrs },
          { value: "ladder", label: t.modelLadder },
        ]}
      />
      {shown.memoryModel === "fsrs" && (
        <Select
          label={t.retention}
          description={t.retentionHint}
          value={String(Math.round(shown.targetRetention * 100))}
          onValueChange={(v) => saveField({ targetRetention: Number(v) / 100 })}
          options={RETENTIONS.map((p) => ({ value: String(p), label: t.retentionOption(p) }))}
        />
      )}
      <fieldset className="flex flex-col gap-1.5">
        <legend className="text-sm font-medium text-fg">{t.restDays}</legend>
        <div className="flex flex-wrap gap-1.5">
          {WEEKDAYS.map((day) => (
            <Chip key={day} selected={shown.restDays.includes(day)} onSelectedChange={() => toggleRest(day)}>{t.weekday(day)}</Chip>
          ))}
        </div>
        <p className="text-xs text-fg-muted">{t.restDaysHint}</p>
      </fieldset>
      {isWitems() ? (
        <p className="text-xs text-fg-muted">{t.reminderWitems}</p>
      ) : (
        <Select
          label={t.reminder}
          description={t.reminderHint}
          value={String(shown.reminderHour)}
          onValueChange={(v) => saveField({ reminderHour: Number(v) })}
          options={[
            { value: "-1", label: t.reminderOff },
            ...HOURS.map((h) => ({ value: String(h), label: t.reminderAt(h) })),
          ]}
        />
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
      <Select
        label={t.fontSize}
        value={ui.fontSize}
        onValueChange={(v) => setUiPrefs({ fontSize: v as FontSize })}
        options={[
          { value: "sm", label: t.fontSm },
          { value: "md", label: t.fontMd },
          { value: "lg", label: t.fontLg },
        ]}
      />
      <Select
        label={t.lineWidth}
        value={ui.lineWidth}
        onValueChange={(v) => setUiPrefs({ lineWidth: v as LineWidth })}
        options={[
          { value: "narrow", label: t.widthNarrow },
          { value: "normal", label: t.widthNormal },
          { value: "wide", label: t.widthWide },
        ]}
      />
      <div>
        <Switch label={t.shortcuts} checked={ui.shortcuts} onCheckedChange={(checked) => setUiPrefs({ shortcuts: checked })} />
        <p className="text-xs text-fg-muted">{t.shortcutsHint}</p>
      </div>
      <p className="text-xs text-fg-muted">{t.deviceHint}</p>
    </div>
  );
}
