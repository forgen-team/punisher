
import { useRef, useState, type KeyboardEvent } from "react";
import { Button, Textarea } from "@/src/shared/ui/kit";
import { describeCheckError, useCheckAnswer, useCheckReveal } from "@/src/entities/punisher";
import { CHECK_MAX_ANSWER_CHARS, type CheckDetail } from "@/src/shared/api/punisher";
import { canSend, showReveal } from "../lib/checkView";

/**
 * 하단 고정 입력창 — 하단 탭 위(--app-bottom-offset)에 붙어 휴대폰에서도 본문을 읽으며 답한다.
 * Enter 는 줄바꿈, ⌘/Ctrl+Enter 로 보낸다(질문 해설의 이어서 묻기와 같다). '그냥 알려줘'는 서버가 허용할 때만.
 */
export function CheckComposer({ detail }: { detail: CheckDetail }) {
  const id = detail.session.id;
  const [text, setText] = useState("");
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const answer = useCheckAnswer(id);
  const reveal = useCheckReveal(id);
  const busy = answer.isPending || reveal.isPending;
  const sendable = canSend(detail, text) && !busy;

  function send() {
    if (!sendable) { return; }
    answer.mutate(text.trim(), {
      onSuccess: () => {
        setText("");
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
    <div className="sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 flex flex-col gap-2 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0">
      {error && (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(error)}
        </p>
      )}
      <Textarea
        ref={inputRef}
        aria-label="답"
        value={text}
        maxLength={CHECK_MAX_ANSWER_CHARS}
        rows={2}
        onChange={(event) => setText(event.target.value)}
        onKeyDown={handleKeyDown}
      />
      <div className="flex items-center justify-end gap-2">
        {showReveal(detail) && (
          <Button variant="ghost" onClick={() => reveal.mutate()} disabled={busy} loading={reveal.isPending}>
            그냥 알려줘
          </Button>
        )}
        <Button onClick={send} disabled={!sendable} loading={answer.isPending}>
          보내기
        </Button>
      </div>
    </div>
  );
}
