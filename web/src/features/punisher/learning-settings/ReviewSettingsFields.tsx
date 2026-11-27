import { useState, type FocusEvent, type KeyboardEvent } from "react";
import { Chip, ErrorState, ListSkeleton, TextField } from "@forgen-team/ui";
import {
  MASTERY_LABELS,
  REVIEW_GRADER_LABELS,
  describeSettingsError,
  useLearningSettings,
  useSaveLearningSettings,
  useSaveStudyPrefs,
  useStudyPrefs,
} from "@/src/entities/punisher";
import {
  describePunisherError,
  type LearningSettings,
  type LearningSettingsPatch,
  type MemoryModel,
  type ReviewGrader,
  type StudyPrefs,
  type StudyPrefsPatch,
} from "@/src/shared/api/punisher";
import { isWitems } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { messages, studyMessages } from "./i18n";
import { MASTERY_TARGET_OPTIONS, REVIEW_MINUTE_OPTIONS, formatIntervals, parseIntervals } from "./reviewSettings";
import { CONTROL_NO_LABEL_GAP, SelectRow, SettingRow, SwitchRow } from "./SettingRow";
import { useFieldSaves } from "./useFieldSaves";

const GRADERS: ReviewGrader[] = ["self", "llm"];
const RETENTIONS = [80, 85, 90, 95] as const;
const HOURS = Array.from({ length: 24 }, (_, h) => h);
const WEEKDAYS = [0, 1, 2, 3, 4, 5, 6] as const;

/** 복습 간격(일) — 쉼표로 적고, 칸을 벗어나거나 Enter 로 저장(바뀌었고 형식이 맞을 때만). */
function IntervalsField({ value, onSave, saved, serverError }: { value: number[]; onSave: (days: number[]) => void; saved: boolean; serverError?: string }) {
  const t = useT(messages);
  const savedText = formatIntervals(value);
  const [text, setText] = useState(savedText);
  const [localError, setLocalError] = useState<string | undefined>();
  const [base, setBase] = useState(savedText);
  if (base !== savedText) {
    setBase(savedText);
    setText(savedText);
  }

  function commit(event: FocusEvent<HTMLInputElement> | KeyboardEvent<HTMLInputElement>) {
    const parsed = parseIntervals(event.currentTarget.value);
    if ("error" in parsed) {
      setLocalError(parsed.error);
      return;
    }
    setLocalError(undefined);
    if (formatIntervals(parsed.days) !== savedText) { onSave(parsed.days); }
  }

  const error = localError ?? serverError;
  return (
    <SettingRow label={t.reviewIntervals} saved={saved} error={error}>
      <div className={`w-36 ${CONTROL_NO_LABEL_GAP}`}>
        <TextField
          label={<span className="sr-only">{t.reviewIntervals}</span>}
          invalid={Boolean(error)}
          value={text}
          onValueChange={({ value: next }) => setText(next)}
          inputProps={{
            inputMode: "numeric",
            onBlur: commit,
            onKeyDown: (event) => {
              if (event.key === "Enter" && !event.nativeEvent.isComposing) { commit(event); }
            },
          }}
        />
      </div>
    </SettingRow>
  );
}

interface RowsProps {
  settings: LearningSettings;
  prefs: StudyPrefs;
  learning: ReturnType<typeof useFieldSaves<LearningSettingsPatch>>;
  study: ReturnType<typeof useFieldSaves<StudyPrefsPatch>>;
}

/** 복습 9항목을 시안 순서대로 — 숙달 기준 · 간격 방식 · 간격(일) · 목표 유지율 · 하루 복습 시간 · 자동 조절 · 채점 방식 · 쉬는 요일 · 알림. */
function ReviewRows({ settings, prefs, learning, study }: RowsProps) {
  const t = useT(messages);
  const ts = useT(studyMessages);
  const minuteOptions = REVIEW_MINUTE_OPTIONS.includes(settings.reviewMinutes)
    ? REVIEW_MINUTE_OPTIONS
    : [...REVIEW_MINUTE_OPTIONS, settings.reviewMinutes].sort((a, b) => a - b);
  const lrow = (field: keyof LearningSettingsPatch) => ({ saved: learning.savedKey === field, error: learning.errors[field] });
  const srow = (field: keyof StudyPrefsPatch) => ({ saved: study.savedKey === field, error: study.errors[field] });
  const toggleRest = (day: number) => {
    const set = new Set(prefs.restDays);
    if (set.has(day)) { set.delete(day); } else if (set.size < 6) { set.add(day); }
    study.saveField({ restDays: [...set].sort((a, b) => a - b) });
  };

  return (
    <div className="flex flex-col">
      <SelectRow
        label={t.masteryTarget}
        {...lrow("masteryTarget")}
        value={String(settings.masteryTarget)}
        onChange={(v) => learning.saveField({ masteryTarget: Number(v) })}
        options={MASTERY_TARGET_OPTIONS.map((level) => ({ value: String(level), label: t.masteryOption(level, MASTERY_LABELS[level]) }))}
      />
      <SelectRow
        label={ts.memoryModel}
        hint={ts.memoryModelHint}
        {...srow("memoryModel")}
        value={prefs.memoryModel}
        onChange={(v) => study.saveField({ memoryModel: v as MemoryModel })}
        options={[
          { value: "fsrs", label: ts.modelFsrs },
          { value: "ladder", label: ts.modelLadder },
        ]}
      />
      <IntervalsField
        value={settings.reviewIntervals ?? []}
        onSave={(days) => learning.saveField({ reviewIntervals: days })}
        saved={learning.savedKey === "reviewIntervals"}
        serverError={learning.errors.reviewIntervals}
      />
      {prefs.memoryModel === "fsrs" && (
        <SelectRow
          label={ts.retention}
          hint={ts.retentionHint}
          {...srow("targetRetention")}
          value={String(Math.round(prefs.targetRetention * 100))}
          onChange={(v) => study.saveField({ targetRetention: Number(v) / 100 })}
          options={RETENTIONS.map((p) => ({ value: String(p), label: ts.retentionOption(p) }))}
        />
      )}
      <SelectRow
        label={t.reviewMinutes}
        {...lrow("reviewMinutes")}
        value={String(settings.reviewMinutes)}
        onChange={(v) => learning.saveField({ reviewMinutes: Number(v) })}
        options={minuteOptions.map((m) => ({ value: String(m), label: t.minutes(m) }))}
      />
      <SwitchRow
        label={t.adaptiveIntervals}
        {...lrow("adaptiveIntervals")}
        checked={settings.adaptiveIntervals ?? true}
        onChange={(checked) => learning.saveField({ adaptiveIntervals: checked })}
      />
      <SelectRow
        label={t.reviewGrader}
        {...lrow("reviewGrader")}
        value={settings.reviewGrader}
        onChange={(v) => learning.saveField({ reviewGrader: v as ReviewGrader })}
        options={GRADERS.map((g) => ({ value: g, label: REVIEW_GRADER_LABELS[g] }))}
      />
      <SettingRow label={ts.restDays} hint={ts.restDaysHint} {...srow("restDays")}>
        <div role="group" aria-label={ts.restDays} className="flex flex-wrap justify-end gap-2">
          {WEEKDAYS.map((day) => (
            <Chip key={day} selected={prefs.restDays.includes(day)} onSelectedChange={() => toggleRest(day)}>{ts.weekday(day)}</Chip>
          ))}
        </div>
      </SettingRow>
      {isWitems() ? (
        <SettingRow label={ts.reminder}>
          <span className="max-w-64 text-right text-xs text-fg-muted">{ts.reminderWitems}</span>
        </SettingRow>
      ) : (
        <SelectRow
          label={ts.reminder}
          hint={ts.reminderHint}
          {...srow("reminderHour")}
          value={String(prefs.reminderHour)}
          onChange={(v) => study.saveField({ reminderHour: Number(v) })}
          options={[
            { value: "-1", label: ts.reminderOff },
            ...HOURS.map((h) => ({ value: String(h), label: ts.reminderAt(h) })),
          ]}
        />
      )}
    </div>
  );
}

/**
 * 설정 › 복습 구역(9항목 한 구역). 값은 두 곳에서 온다 — 학습 설정(PUT /me/settings: 숙달 기준 · 간격(일) · 하루 복습 시간 ·
 * 자동 조절 · 채점 방식)과 복습 기억 설정(PUT /me/study-prefs: 간격 방식 · 목표 유지율 · 쉬는 요일 · 알림). 둘 다 바꾸면 그 칸만 바로
 * 저장하고(숫자 칸은 칸을 떠날 때), 그 자리에 "저장했어요" 를 잠깐 보인다. 저장 중에도 다른 칸을 바꿀 수 있다.
 * 목표 유지율은 기억 모델(FSRS)일 때만 뜻이 있어 그때만 보인다.
 */
export function ReviewSettingsFields() {
  const settings = useLearningSettings();
  const prefs = useStudyPrefs();
  const saveSettings = useSaveLearningSettings();
  const savePrefs = useSaveStudyPrefs();
  const learning = useFieldSaves<LearningSettingsPatch>(saveSettings.mutateAsync, describeSettingsError);
  const study = useFieldSaves<StudyPrefsPatch>(savePrefs.mutateAsync, describePunisherError);

  if (settings.isLoading || prefs.isLoading) { return <ListSkeleton rows={9} />; }
  if (settings.error || prefs.error || !settings.data || !prefs.data) {
    return (
      <ErrorState
        title={describePunisherError(settings.error ?? prefs.error)}
        onRetry={() => { void settings.refetch(); void prefs.refetch(); }}
      />
    );
  }
  return (
    <ReviewRows
      settings={{ ...settings.data.settings, ...learning.unsaved }}
      prefs={{ ...prefs.data, ...study.unsaved }}
      learning={learning}
      study={study}
    />
  );
}
