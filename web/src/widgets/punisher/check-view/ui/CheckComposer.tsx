import { useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Button, TextField } from "@forgen-team/ui";
import {
  SELF_VERDICT_LABELS_BY_LOCALE,
  SELF_VERDICT_ORDER,
  describeCheckError,
  useCheckAnswer,
  useCheckReveal,
  useCheckSelfGrade,
  useCheckSkip,
} from "@/src/entities/punisher";
import { CHECK_MAX_ANSWER_CHARS, type CheckDetail, type Confidence } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { ConfidencePicker } from "./ConfidencePicker";
import { asksConfidence, canSend, showReveal, showSkip } from "../lib/checkView";
import { useStudyShortcuts } from "@/src/shared/lib/shortcuts";

const FOOTER_CLASS =
  "sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 flex flex-col gap-2 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0";

/** 고정 Footer — 하단 탭 위(--app-bottom-offset)에 붙는다. 안은 한 줄(왼쪽 보조 … 오른쪽 주). */
export function StepFooter({ children, error }: { children: ReactNode; error?: unknown }) {
  return (
    <div className={FOOTER_CLASS}>
      {error ? (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(error)}
        </p>
      ) : null}
      <div className="flex items-center gap-2">{children}</div>
    </div>
  );
}

/** 자기 채점 Footer — 모범답·기준을 본 뒤 맞음 / 부분 / 틀림. 단축키 1·2·3(설정에서 끌 수 있음)은 버튼 순서와 같다. */
export function SelfGradeFooter({ id }: { id: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const grade = useCheckSelfGrade(id);
  const pick = (index: number) => () => {
    const verdict = SELF_VERDICT_ORDER[index];
    if (verdict && !grade.isPending) { grade.mutate(verdict); }
  };
  const keys = useStudyShortcuts({ "1": pick(0), "2": pick(1), "3": pick(2) });
  return (
    <StepFooter error={grade.error}>
      <div role="group" aria-label={t.selfGrade} className="grid flex-1 grid-cols-3 gap-2">
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
    </StepFooter>
  );
}

/** 답 입력 상태 — 입력란(본문)과 Footer 버튼이 함께 쓴다. */
export function useAnswerForm(detail: CheckDetail) {
  const id = detail.session.id;
  const [text, setText] = useState("");
  const [confidence, setConfidence] = useState<Confidence>();
  // ui TextField 의 inputRef 는 input·textarea 겸용 타입이다.
  const inputRef = useRef<HTMLInputElement & HTMLTextAreaElement>(null);
  const answer = useCheckAnswer(id);
  const reveal = useCheckReveal(id);
  const skip = useCheckSkip(id);
  const busy = answer.isPending || reveal.isPending || skip.isPending;
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

  /** 지금 문항을 건너뛴다 — 성공하면 서버가 다음 문항을 묻는다(입력란은 문항마다 새로 그린다). */
  function skipItem() {
    const itemId = detail.current?.itemId;
    if (!itemId || busy) { return; }
    skip.mutate(itemId);
  }

  return {
    text, setText, confidence, setConfidence, inputRef, answer, reveal, skip, skipItem, busy, sendable, send,
    error: answer.error ?? reveal.error ?? skip.error,
  };
}

export type AnswerForm = ReturnType<typeof useAnswerForm>;

/**
 * 답 입력란 + 확신도 칩(본문, 문항 아래). Enter 는 줄바꿈, ⌘/Ctrl+Enter 로 보낸다(질문 해설의 이어서 묻기와 같다).
 */
export function AnswerFields({ detail, form }: { detail: CheckDetail; form: AnswerForm }) {
  const t = useT(messages);
  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey) && !event.nativeEvent.isComposing) {
      event.preventDefault();
      form.send();
    }
  }
  return (
    <div className="flex flex-col gap-3">
      <TextField
        multiline
        inputRef={form.inputRef}
        value={form.text}
        onValueChange={({ value }) => form.setText(value)}
        hideCharacterCount
        textareaProps={{ "aria-label": t.answer, maxLength: CHECK_MAX_ANSWER_CHARS, rows: 4, onKeyDown: handleKeyDown }}
      />
      {asksConfidence(detail) && <ConfidencePicker value={form.confidence} onChange={form.setConfidence} />}
    </div>
  );
}

/** 답 Footer — `[건너뛰기] … [그냥 알려줘][보내기]`. 건너뛰기·그냥 알려줘는 서버가 허용할 때만. */
export function AnswerFooter({ detail, form }: { detail: CheckDetail; form: AnswerForm }) {
  const t = useT(messages);
  return (
    <StepFooter error={form.error}>
      {showSkip(detail) && (
        <Button variant="ghost" onClick={form.skipItem} disabled={form.busy} loading={form.skip.isPending}>
          {t.skip}
        </Button>
      )}
      <span className="flex-1" />
      {showReveal(detail) && (
        <Button variant="ghost" onClick={() => form.reveal.mutate()} disabled={form.busy} loading={form.reveal.isPending}>
          {t.reveal}
        </Button>
      )}
      <Button variant="primary" onClick={form.send} disabled={!form.sendable} loading={form.answer.isPending}>
        {t.send}
      </Button>
    </StepFooter>
  );
}
