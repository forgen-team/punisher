import { useRef, useState, type KeyboardEvent } from "react";
import { Button, Textarea } from "@/src/shared/ui/kit";
import {
  SELF_VERDICT_LABELS_BY_LOCALE,
  SELF_VERDICT_ORDER,
  describeCheckError,
  useCheckAnswer,
  useCheckReveal,
  useCheckSelfGrade,
} from "@/src/entities/punisher";
import { CHECK_MAX_ANSWER_CHARS, type CheckDetail, type Confidence } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { ConfidencePicker } from "./ConfidencePicker";
import { asksConfidence, canSend, selfGradeStep, showReveal } from "../lib/checkView";

const COMPOSER_CLASS =
  "sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 flex flex-col gap-2 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0";

/** 자기 채점 — 모범답·기준을 본 뒤 맞음 / 부분 / 틀림. */
function SelfGradeBar({ id }: { id: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const grade = useCheckSelfGrade(id);
  return (
    <div className={COMPOSER_CLASS}>
      {grade.error && (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(grade.error)}
        </p>
      )}
      <div role="group" aria-label={t.selfGrade} className="grid grid-cols-3 gap-2">
        {SELF_VERDICT_ORDER.map((verdict) => (
          <Button
            key={verdict}
            variant="secondary"
            disabled={grade.isPending}
            loading={grade.isPending && grade.variables === verdict}
            onClick={() => grade.mutate(verdict)}
          >
            {SELF_VERDICT_LABELS_BY_LOCALE[locale][verdict]}
          </Button>
        ))}
      </div>
    </div>
  );
}

/** 하단 고정 입력창 — 자기 채점에서 답을 보냈으면 채점 버튼으로 바뀐다. */
export function CheckComposer({ detail }: { detail: CheckDetail }) {
  if (selfGradeStep(detail) === "grade") { return <SelfGradeBar key={detail.current?.itemId} id={detail.session.id} />; }
  return <AnswerComposer detail={detail} />;
}

/**
 * 답 입력창 — 하단 탭 위(--app-bottom-offset)에 붙어 휴대폰에서도 본문을 읽으며 답한다.
 * Enter 는 줄바꿈, ⌘/Ctrl+Enter 로 보낸다(질문 해설의 이어서 묻기와 같다). '그냥 알려줘'는 서버가 허용할 때만.
 */
function AnswerComposer({ detail }: { detail: CheckDetail }) {
  const t = useT(messages);
  const id = detail.session.id;
  const [text, setText] = useState("");
  const [confidence, setConfidence] = useState<Confidence>();
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const answer = useCheckAnswer(id);
  const reveal = useCheckReveal(id);
  const busy = answer.isPending || reveal.isPending;
  const sendable = canSend(detail, text) && !busy;

  function send() {
    if (!sendable) { return; }
    answer.mutate({ text: text.trim(), confidence: asksConfidence(detail) ? confidence : undefined }, {
      onSuccess: () => {
        setText("");
        setConfidence(undefined);
        inputRef.current?.focus();
      },
    });
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey) && !event.nativeEvent.isComposing) {
      event.preventDefault();
      send();
    }
  }

  const error = answer.error ?? reveal.error;
  return (
    <div className={COMPOSER_CLASS}>
      {error && (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(error)}
        </p>
      )}
      {asksConfidence(detail) && <ConfidencePicker value={confidence} onChange={setConfidence} />}
      <Textarea
        ref={inputRef}
        aria-label={t.answer}
        value={text}
        maxLength={CHECK_MAX_ANSWER_CHARS}
        rows={2}
        onChange={(event) => setText(event.target.value)}
        onKeyDown={handleKeyDown}
      />
      <div className="flex items-center justify-end gap-2">
        {showReveal(detail) && (
          <Button variant="ghost" onClick={() => reveal.mutate()} disabled={busy} loading={reveal.isPending}>
            {t.reveal}
          </Button>
        )}
        <Button onClick={send} disabled={!sendable} loading={answer.isPending}>
          {t.send}
        </Button>
      </div>
    </div>
  );
}
