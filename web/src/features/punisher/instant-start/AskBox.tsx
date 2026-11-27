import { useRouter } from "@/src/shared/lib/router";
import { useState, type FormEvent, type KeyboardEvent } from "react";
import { Button, SegmentedControl, SegmentedControlItem, TextField, useToast } from "@forgen-team/ui";
import { useCreateInquiry, useCreateStudyRequest } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useDraftState } from "@/src/shared/lib/drafts";
import type { InstantResponse } from "@/src/shared/api/punisher";
import { ResolutionChoices } from "./ResolutionChoices";
import { inquiryHref, useLaunchInstant } from "./useLaunchInstant";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

type AskMode = "ask" | "topic";

const MAX: Record<AskMode, number> = { ask: 1000, topic: 500 };

/**
 * 오늘 › 묻기·새 주제 — 전환 2 [질문하기 | 주제 공부하기] + 입력 한 칸 + [보내기].
 * 질문하기: 질문 해설을 만들고 해설 화면으로(1~3분). 긴 상황 설명이라 Enter 는 줄바꿈, ⌘/Ctrl+Enter 로 보낸다.
 * 주제 공부하기: 지도에서 개념을 찾아 그 자리에서 후보를 고른다 → 강의 시작, 또는 [나중에 공부](공부할 주제에 남김).
 */
export function AskBox() {
  const router = useRouter();
  const t = useT(messages);
  const toast = useToast();
  const [mode, setMode] = useState<AskMode>("ask");
  // 쓰던 글은 이 기기에 보관한다 — 다른 화면에 갔다 와도 남고, 보내면 지운다(점검 17).
  const draft = useDraftState("instant-query", "");
  const query = draft.value;
  const [submitted, setSubmitted] = useState("");
  const [result, setResult] = useState<InstantResponse | null>(null);
  // 해설 화면으로 넘어가는 동안 한 번 더 보내지 않게(mutation 이 끝난 뒤에도 이동 전까지 막는다).
  const [navigating, setNavigating] = useState(false);
  const instant = useLaunchInstant();
  const inquiry = useCreateInquiry();
  const later = useCreateStudyRequest();
  const isPending = instant.isPending || inquiry.isPending || navigating;
  const error = mode === "ask" ? inquiry.error : (instant.error ?? later.error);
  const max = MAX[mode];
  // 모드를 바꿔도 쓴 글은 그대로 둔다 — 한도를 넘으면 경고하고 보내기만 막는다.
  const tooLong = query.trim().length > max;

  async function submit() {
    const trimmed = query.trim();
    if (!trimmed || isPending || trimmed.length > max) { return; }
    setResult(null);
    setSubmitted(trimmed);
    try {
      if (mode === "ask") {
        const created = await inquiry.mutateAsync(trimmed);
        draft.clear();
        setNavigating(true);
        router.push(inquiryHref(created.id));
        return;
      }
      setResult(await instant.launch({ query: trimmed }));
      draft.clear();
    } catch {
      // 오류는 아래 <InlineError error={error}> 가 표시한다.
      setResult(null);
    }
  }

  async function pick(conceptId: string) {
    try {
      setResult(await instant.launch({ conceptId, query: submitted }));
    } catch {
      // 선택지를 유지하고 오류만 표시한다.
      return;
    }
  }

  function studyLater() {
    if (!submitted || later.isPending) { return; }
    later.mutate(submitted, {
      onSuccess: () => {
        setResult(null);
        toast.show(t.savedForLater);
      },
    });
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    void submit();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key !== "Enter" || event.nativeEvent.isComposing) { return; }
    const send = mode === "ask" ? event.metaKey || event.ctrlKey : !event.shiftKey;
    if (send) {
      event.preventDefault();
      void submit();
    }
  }

  function pickMode(next: AskMode) {
    setMode(next);
    setResult(null);
    inquiry.reset();
    instant.reset();
    later.reset();
  }

  const label = mode === "ask" ? t.askLabel : t.topicLabel;
  return (
    <div className="flex flex-col gap-3">
      <SegmentedControl aria-label={t.modeGroup} value={mode} onValueChange={(v) => pickMode(v as AskMode)}>
        <SegmentedControlItem value="ask">{t.askLabel}</SegmentedControlItem>
        <SegmentedControlItem value="topic">{t.topicLabel}</SegmentedControlItem>
      </SegmentedControl>
      <form onSubmit={handleSubmit} className="flex flex-col gap-2">
        <TextField
          multiline
          value={query}
          onValueChange={({ value }) => draft.setValue(value)}
          placeholder={mode === "ask" ? t.askPlaceholder : t.topicPlaceholder}
          textareaProps={{
            "aria-label": label,
            maxLength: MAX.ask,
            rows: 3,
            enterKeyHint: mode === "ask" ? "enter" : "send",
            onKeyDown: handleKeyDown,
            className: "resize-y",
          }}
        />
        <div className="flex items-center justify-end gap-3">
          {tooLong && (
            <span role="alert" className="text-xs font-medium tabular-nums text-danger">
              {t.tooLong(max)}
            </span>
          )}
          <Button type="submit" variant="primary" size="sm" loading={isPending} disabled={!query.trim() || tooLong}>
            {t.send}
          </Button>
        </div>
      </form>
      <div aria-live="polite" className="flex flex-col gap-2 empty:hidden">
        <InlineError error={error} />
        {mode === "topic" && result && (
          <ResolutionChoices response={result} disabled={isPending || later.isPending} onPick={pick} onLater={studyLater} />
        )}
      </div>
    </div>
  );
}
