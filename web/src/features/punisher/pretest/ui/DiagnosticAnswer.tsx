import { useState, type FormEvent } from "react";
import { Button, TextField } from "@forgen-team/ui";
import { describeCheckError, useCheckAnswer, useCheckChoice } from "@/src/entities/punisher";
import { CHECK_MAX_ANSWER_CHARS, type CheckDetail, type CheckItem } from "@/src/shared/api/punisher";
import { choicesOf } from "../lib/diagnostic";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 진단 문항의 답 — 보기 버튼(choice) 또는 한 줄 예측 + '모르겠어요'. 고르면 바로 기록되고 다음 문항으로 간다. */
export function DiagnosticAnswer({ detail, item }: { detail: CheckDetail; item: CheckItem }) {
  const t = useT(messages);
  const id = detail.session.id;
  const choice = useCheckChoice(id);
  const answer = useCheckAnswer(id);
  const [text, setText] = useState("");
  const [picked, setPicked] = useState<number | null>(null);
  const busy = choice.isPending || answer.isPending;
  const choices = choicesOf(item);
  const error = choice.error ?? answer.error;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmed = text.trim();
    if (!trimmed || busy) { return; }
    answer.mutate({ text: trimmed }, { onSuccess: () => setText("") });
  }

  return (
    <div className="flex flex-col gap-2">
      {choices.length > 0 ? (
        <div role="group" aria-label={t.choices} className="flex flex-col gap-2">
          {choices.map((label, index) => (
            <Button
              key={`${item.id}-${index}`}
              variant="secondary"
              className="h-auto min-h-touch justify-start whitespace-normal py-2 text-left"
              disabled={busy}
              loading={choice.isPending && picked === index}
              onClick={() => {
                setPicked(index);
                choice.mutate({ choice: index });
              }}
            >
              {label}
            </Button>
          ))}
        </div>
      ) : (
        <form onSubmit={submit} className="flex gap-2">
          <TextField
            value={text}
            disabled={busy}
            onValueChange={({ value }) => setText(value)}
            inputProps={{ "aria-label": t.answer, maxLength: CHECK_MAX_ANSWER_CHARS }}
            className="min-w-0 flex-1"
          />
          <Button type="submit" variant="primary" disabled={!text.trim() || busy} loading={answer.isPending}>
            {t.send}
          </Button>
        </form>
      )}
      <div>
        <Button variant="ghost" size="sm" disabled={busy} onClick={() => {
          setPicked(null);
          choice.mutate({ dontKnow: true });
        }}>
          {t.dontKnow}
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(error)}
        </p>
      )}
    </div>
  );
}
