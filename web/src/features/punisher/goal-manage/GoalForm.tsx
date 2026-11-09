
import { useState, type FormEvent } from "react";
import { Button, Field, FieldInput } from "@/src/shared/ui/kit";
import { useCreateGoal } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { ConceptCombobox } from "../concept-combobox";

/** 새 목표: 이름 + 목표 개념(복수). */
export function GoalForm() {
  const create = useCreateGoal();
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
      <Field label="목표 이름">
        <FieldInput
          value={title}
          maxLength={200}
          onChange={(event) => setTitle(event.target.value)}
          placeholder="예: Pod 장애 이해"
        />
      </Field>
      <ConceptCombobox label="목표 개념" value={targets} onChange={setTargets} />
      <Button
        type="submit"
        variant="secondary"
        loading={create.isPending}
        disabled={!title.trim() || targets.length === 0}
        className="self-start"
      >
        목표 추가
      </Button>
      <div aria-live="polite">
        <InlineError error={create.error} />
      </div>
    </form>
  );
}
