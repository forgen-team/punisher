
import { useState, type FormEvent } from "react";
import { Button, TextField } from "@forgen-team/ui";
import { useCreateGoal } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { ConceptCombobox } from "../concept-combobox";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 새 목표: 이름 + 목표 개념(복수). */
export function GoalForm() {
  const create = useCreateGoal();
  const t = useT(messages);
  const [title, setTitle] = useState("");
  const [targets, setTargets] = useState<string[]>([]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!title.trim() || targets.length === 0 || create.isPending) { return; }
    create.mutate(
      { title: title.trim(), targetConceptIds: targets },
      {
        onSuccess: () => {
          setTitle("");
          setTargets([]);
        },
      },
    );
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3">
      <TextField
        label={t.goalName}
        value={title}
        onValueChange={({ value }) => setTitle(value)}
        placeholder={t.goalNamePlaceholder}
        inputProps={{ maxLength: 200 }}
      />
      <ConceptCombobox label={t.targetConcepts} value={targets} onChange={setTargets} />
      <Button
        type="submit"
        variant="secondary"
        loading={create.isPending}
        disabled={!title.trim() || targets.length === 0}
        className="self-start"
      >
        {t.addGoal}
      </Button>
      <div aria-live="polite">
        <InlineError error={create.error} />
      </div>
    </form>
  );
}
