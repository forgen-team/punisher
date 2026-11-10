import { forwardRef, useImperativeHandle, useRef, useState, type KeyboardEvent } from "react";
import { X } from "lucide-react";
import { Button, IconButton, Section, Textarea } from "@/src/shared/ui/kit";
import { useCreateFollowup } from "@/src/entities/punisher";
import { INQUIRY_MAX_FOLLOWUPS, INQUIRY_MAX_FOLLOWUP_CHARS } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";

interface FollowupComposerProps {
  inquiryId: string;
  used: number;
  /** 답을 쓰는 중이면 끝날 때까지 보낼 수 없다 */
  busy: boolean;
  quote: string;
  onClearQuote: () => void;
}

/** 해설 아래 '이어서 묻기' — 문서 흐름의 끝에 둔다(하단 고정이 아니라 본문을 가리지 않음). Enter 는 줄바꿈, ⌘/Ctrl+Enter 로 보낸다. */
export const FollowupComposer = forwardRef<HTMLTextAreaElement, FollowupComposerProps>(function FollowupComposer(
  { inquiryId, used, busy, quote, onClearQuote },
  ref,
) {
  const [text, setText] = useState("");
  const inputRef = useRef<HTMLTextAreaElement>(null);
  useImperativeHandle(ref, () => inputRef.current as HTMLTextAreaElement);
  const create = useCreateFollowup(inquiryId);
  const left = INQUIRY_MAX_FOLLOWUPS - used;
  const disabled = busy || create.isPending || !text.trim() || left <= 0;

  function send() {
    if (disabled) { return; }
    create.mutate(
      { question: text.trim(), quote: quote || undefined },
      {
        onSuccess: () => {
          setText("");
          onClearQuote();
          // 버튼이 비활성으로 바뀌어도 키보드 위치를 잃지 않게.
          inputRef.current?.focus();
        },
      },
    );
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey) && !event.nativeEvent.isComposing) {
      event.preventDefault();
      send();
    }
  }

  if (left <= 0) {
    return (
      <p className="mt-10 break-keep rounded-md bg-canvas-subtle px-3 py-2 text-sm text-fg-secondary">
        이 질문에서 이어서 물을 수 있는 횟수를 다 썼어요. 학습 홈에서 새 질문으로 물어봐 주세요.
      </p>
    );
  }

  return (
    <Section title="이어서 묻기" description={`본문에서 문장을 고르면 그 부분을 인용해 물을 수 있어요 · ${left}번 더 물을 수 있어요`} className="mt-10">
      <div className="flex flex-col gap-2">
        {quote && (
          <div className="flex items-start gap-2 rounded-md bg-canvas-subtle py-1 pl-3 pr-1">
            <blockquote className="line-clamp-3 min-w-0 flex-1 break-keep py-1 text-sm text-fg-secondary">{quote}</blockquote>
            <IconButton size="sm" aria-label="고른 구절 빼기" onClick={onClearQuote}>
              <X aria-hidden="true" />
            </IconButton>
          </div>
        )}
        <Textarea
          ref={inputRef}
          aria-label="이어서 물어볼 내용"
          value={text}
          maxLength={INQUIRY_MAX_FOLLOWUP_CHARS}
          rows={3}
          placeholder={quote ? "고른 구절에서 무엇이 더 궁금한가요?" : "예) 풀 크기는 어떻게 정하나요?"}
          onChange={(event) => setText(event.target.value)}
          onKeyDown={handleKeyDown}
        />
        <div className="flex items-center justify-between gap-3">
          <span className="text-xs tabular-nums text-fg-muted">
            {text.length}/{INQUIRY_MAX_FOLLOWUP_CHARS}
            <span className="[@media(pointer:coarse)]:hidden"> · ⌘/Ctrl+Enter로 보내요</span>
          </span>
          <Button onClick={send} disabled={disabled} loading={create.isPending}>
            물어보기
          </Button>
        </div>
        <div aria-live="polite" className="empty:hidden">
          <InlineError error={create.error} />
        </div>
      </div>
    </Section>
  );
});
