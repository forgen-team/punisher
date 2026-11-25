import { useState, type FocusEvent, type KeyboardEvent } from "react";
import { Field, FieldInput, Select, Switch } from "@/src/shared/ui/kit";
import { MASTERY_LABELS, REVIEW_GRADER_LABELS } from "@/src/entities/punisher";
import type { LearningSettings, LearningSettingsPatch, ReviewGrader } from "@/src/shared/api/punisher";
import { MASTERY_TARGET_OPTIONS, REVIEW_MINUTE_OPTIONS, formatIntervals, parseIntervals, type SettingsField } from "./reviewSettings";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface ReviewSettingsFieldsProps {
  settings: LearningSettings;
  onSave: (patch: LearningSettingsPatch) => void;
  /** 그 필드를 저장하다 난 오류(서버 400 문구) */
  fieldError: (field: SettingsField) => string | undefined;
}

const GRADERS: ReviewGrader[] = ["self", "llm"];

/** 복습 간격 — 쉼표로 적고, 칸을 벗어나거나 Enter 로 저장(바뀌었고 형식이 맞을 때만). */
function IntervalsField({ value, onSave, serverError }: { value: number[]; onSave: (days: number[]) => void; serverError?: string }) {
  const t = useT(messages);
  const saved = formatIntervals(value);
  const [text, setText] = useState(saved);
  const [localError, setLocalError] = useState<string | undefined>();
  const [base, setBase] = useState(saved);
  if (base !== saved) {
    setBase(saved);
    setText(saved);
  }

  function commit(event: FocusEvent<HTMLInputElement> | KeyboardEvent<HTMLInputElement>) {
    const parsed = parseIntervals(event.currentTarget.value);
    if ("error" in parsed) {
      setLocalError(parsed.error);
      return;
    }
    setLocalError(undefined);
    if (formatIntervals(parsed.days) !== saved) { onSave(parsed.days); }
  }

  return (
    <Field label={t.reviewIntervals} error={localError ?? serverError}>
      <FieldInput
        inputMode="numeric"
        value={text}
        onChange={(event) => setText(event.target.value)}
        onBlur={commit}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.nativeEvent.isComposing) { commit(event); }
        }}
      />
    </Field>
  );
}

/** 설정 > 학습의 숙달·복습 항목 — 숙달 기준 · 복습 간격 · 하루 복습 시간 · 복습 채점 방식(부분 저장). */
export function ReviewSettingsFields({ settings, onSave, fieldError }: ReviewSettingsFieldsProps) {
  const t = useT(messages);
  const minuteOptions = REVIEW_MINUTE_OPTIONS.includes(settings.reviewMinutes)
    ? REVIEW_MINUTE_OPTIONS
    : [...REVIEW_MINUTE_OPTIONS, settings.reviewMinutes].sort((a, b) => a - b);
  return (
    <>
      <Field label={t.masteryTarget} error={fieldError("masteryTarget")}>
        <Select value={String(settings.masteryTarget)} onChange={(event) => onSave({ masteryTarget: Number(event.target.value) })}>
          {MASTERY_TARGET_OPTIONS.map((level) => (
            <option key={level} value={level}>
              {t.masteryOption(level, MASTERY_LABELS[level])}
            </option>
          ))}
        </Select>
      </Field>
      <IntervalsField
        value={settings.reviewIntervals ?? []}
        onSave={(days) => onSave({ reviewIntervals: days })}
        serverError={fieldError("reviewIntervals")}
      />
      <Field label={t.reviewMinutes} error={fieldError("reviewMinutes")}>
        <Select value={String(settings.reviewMinutes)} onChange={(event) => onSave({ reviewMinutes: Number(event.target.value) })}>
          {minuteOptions.map((m) => (
            <option key={m} value={m}>
              {t.minutes(m)}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t.adaptiveIntervals} error={fieldError("adaptiveIntervals")}>
        <Switch
          checked={settings.adaptiveIntervals ?? true}
          onCheckedChange={(checked) => onSave({ adaptiveIntervals: checked })}
        />
      </Field>
      <Field label={t.reviewGrader} error={fieldError("reviewGrader")}>
        <Select value={settings.reviewGrader} onChange={(event) => onSave({ reviewGrader: event.target.value as ReviewGrader })}>
          {GRADERS.map((g) => (
            <option key={g} value={g}>
              {REVIEW_GRADER_LABELS[g]}
            </option>
          ))}
        </Select>
      </Field>
    </>
  );
}
