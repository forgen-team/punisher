
import { useState, type FormEvent } from "react";
import { Button, Input } from "@/src/shared/ui/kit";
import { describeCheckError, useCheckAnswer, useCheckChoice } from "@/src/entities/punisher";
import { CHECK_MAX_ANSWER_CHARS, type CheckDetail, type CheckItem } from "@/src/shared/api/punisher";
import { choicesOf } from "../lib/diagnostic";
import { DIAGNOSTIC_LABELS } from "./constants";

/** 진단 문항의 답 — 보기 버튼(choice) 또는 한 줄 예측 + '모르겠어요'. 고르면 바로 기록되고 다음 문항으로 간다. */
export function DiagnosticAnswer({ detail, item }: { detail: CheckDetail; item: CheckItem }) {
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
    answer.mutate(trimmed, { onSuccess: () => setText("") });
  }

  return (
    <div className="flex flex-col gap-2">
      {choices.length > 0 ? (
        <div role="group" aria-label="보기" className="flex flex-col gap-2">
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
          <Input
            aria-label={DIAGNOSTIC_LABELS.answer}
            value={text}
            maxLength={CHECK_MAX_ANSWER_CHARS}
            disabled={busy}
            onChange={(event) => setText(event.target.value)}
            className="min-w-0 flex-1"
          />
          <Button type="submit" disabled={!text.trim() || busy} loading={answer.isPending}>
            {DIAGNOSTIC_LABELS.send}
          </Button>
        </form>
      )}
      <div>
        <Button variant="ghost" size="sm" disabled={busy} onClick={() => {
          setPicked(null);
          choice.mutate({ dontKnow: true });
        }}>
          {DIAGNOSTIC_LABELS.dontKnow}
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
