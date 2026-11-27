import { useState, type FormEvent } from "react";
import { Button, DateField, TextField, toISODay, type CalendarDay } from "@forgen-team/ui";
import { useCreateGoal, useSetGoalDeadline } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { ConceptCombobox } from "../concept-combobox";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 목표 추가: 이름 한 줄 + 기한(선택) + 목표 개념(복수 — 서버가 요구한다). */
export function GoalForm() {
  const create = useCreateGoal();
  const setDeadline = useSetGoalDeadline();
  const t = useT(messages);
  const [title, setTitle] = useState("");
  const [due, setDue] = useState<CalendarDay | undefined>(undefined);
  const [targets, setTargets] = useState<string[]>([]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!title.trim() || targets.length === 0 || create.isPending) { return; }
    const dueOn = due ? toISODay(due) : "";
    create.mutate(
      { title: title.trim(), targetConceptIds: targets },
      {
        onSuccess: (goal) => {
          if (dueOn) { setDeadline.mutate({ goalId: goal.id, dueOn }); }
          setTitle("");
          setDue(undefined);
          setTargets([]);
        },
      },
    );
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end gap-2">
        <div className="min-w-0 flex-1 basis-48">
          <TextField
            label={t.goalName}
            value={title}
            onValueChange={({ value }) => setTitle(value)}
            placeholder={t.goalNamePlaceholder}
            inputProps={{ maxLength: 200 }}
          />
        </div>
        <div className="w-44 shrink-0">
          <DateField label={t.deadlineOptional} value={due} onValueChange={setDue} />
        </div>
      </div>
      <ConceptCombobox label={t.targetConcepts} value={targets} onChange={setTargets} />
      <Button
        type="submit"
        variant="primary"
        loading={create.isPending}
        disabled={!title.trim() || targets.length === 0}
        className="self-start"
      >
        {t.addGoal}
      </Button>
      <div aria-live="polite">
        <InlineError error={create.error ?? setDeadline.error} />
      </div>
    </form>
  );
}
