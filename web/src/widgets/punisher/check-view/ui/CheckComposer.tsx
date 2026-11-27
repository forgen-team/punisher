import { useRef, useState, type KeyboardEvent } from "react";
import { Button, TextField } from "@forgen-team/ui";
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
import { useStudyShortcuts } from "@/src/shared/lib/shortcuts";

const COMPOSER_CLASS =
  "sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 flex flex-col gap-2 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0";

/** 자기 채점 — 모범답·기준을 본 뒤 맞음 / 부분 / 틀림. 단축키 1·2·3(설정에서 끌 수 있음)은 버튼 순서와 같다. */
function SelfGradeBar({ id }: { id: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const grade = useCheckSelfGrade(id);
  const pick = (index: number) => () => {
    const verdict = SELF_VERDICT_ORDER[index];
    if (verdict && !grade.isPending) { grade.mutate(verdict); }
  };
  const keys = useStudyShortcuts({ "1": pick(0), "2": pick(1), "3": pick(2) });
  return (
    <div className={COMPOSER_CLASS}>
      {grade.error && (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(grade.error)}
        </p>
      )}
      <div role="group" aria-label={t.selfGrade} className="grid grid-cols-3 gap-2">
        {SELF_VERDICT_ORDER.map((verdict, index) => (
          <Button
            key={verdict}
            variant="secondary"
            disabled={grade.isPending}
            loading={grade.isPending && grade.variables === verdict}
            onClick={() => grade.mutate(verdict)}
            aria-keyshortcuts={keys ? String(index + 1) : undefined}
          >
            {keys && <kbd className="hidden rounded-sm border border-line px-1 text-xs text-fg-muted md:inline">{index + 1}</kbd>}
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
  // ui TextField 의 inputRef 는 input·textarea 겸용 타입이다.
  const inputRef = useRef<HTMLInputElement & HTMLTextAreaElement>(null);
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
      <TextField
        multiline
        inputRef={inputRef}
        value={text}
        onValueChange={({ value }) => setText(value)}
        hideCharacterCount
        textareaProps={{ "aria-label": t.answer, maxLength: CHECK_MAX_ANSWER_CHARS, rows: 2, onKeyDown: handleKeyDown }}
      />
      <div className="flex items-center justify-end gap-2">
        {showReveal(detail) && (
          <Button variant="ghost" onClick={() => reveal.mutate()} disabled={busy} loading={reveal.isPending}>
            {t.reveal}
          </Button>
        )}
        <Button variant="primary" onClick={send} disabled={!sendable} loading={answer.isPending}>
          {t.send}
        </Button>
      </div>
    </div>
  );
}
