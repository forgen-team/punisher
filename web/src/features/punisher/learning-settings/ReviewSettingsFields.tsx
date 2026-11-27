import { useState, type FocusEvent, type KeyboardEvent } from "react";
import { Select, Switch, TextField } from "@forgen-team/ui";
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
    <TextField
      label={t.reviewIntervals}
      errorMessage={localError ?? serverError}
      invalid={Boolean(localError ?? serverError)}
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
      <Select
        label={t.masteryTarget}
        errorMessage={fieldError("masteryTarget")}
        invalid={Boolean(fieldError("masteryTarget"))}
        value={String(settings.masteryTarget)}
        onValueChange={(v) => onSave({ masteryTarget: Number(v) })}
        options={MASTERY_TARGET_OPTIONS.map((level) => ({ value: String(level), label: t.masteryOption(level, MASTERY_LABELS[level]) }))}
      />
      <IntervalsField
        value={settings.reviewIntervals ?? []}
        onSave={(days) => onSave({ reviewIntervals: days })}
        serverError={fieldError("reviewIntervals")}
      />
      <Select
        label={t.reviewMinutes}
        errorMessage={fieldError("reviewMinutes")}
        invalid={Boolean(fieldError("reviewMinutes"))}
        value={String(settings.reviewMinutes)}
        onValueChange={(v) => onSave({ reviewMinutes: Number(v) })}
        options={minuteOptions.map((m) => ({ value: String(m), label: t.minutes(m) }))}
      />
      <div>
        <Switch
          label={t.adaptiveIntervals}
          checked={settings.adaptiveIntervals ?? true}
          onCheckedChange={(checked) => onSave({ adaptiveIntervals: checked })}
        />
        {fieldError("adaptiveIntervals") && (
          <p role="alert" className="text-xs text-danger">{fieldError("adaptiveIntervals")}</p>
        )}
      </div>
      <Select
        label={t.reviewGrader}
        errorMessage={fieldError("reviewGrader")}
        invalid={Boolean(fieldError("reviewGrader"))}
        value={settings.reviewGrader}
        onValueChange={(v) => onSave({ reviewGrader: v as ReviewGrader })}
        options={GRADERS.map((g) => ({ value: g, label: REVIEW_GRADER_LABELS[g] }))}
      />
    </>
  );
}
