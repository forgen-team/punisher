import { useRouter } from "@/src/shared/lib/router";
import { useState, type FormEvent, type KeyboardEvent } from "react";
import { ArrowRight, BookOpen, MessageCircleQuestion } from "lucide-react";
import { IconButton, ProgressCircle, SegmentedControl, SegmentedControlItem, TextField } from "@forgen-team/ui";
import { cn } from "@/src/shared/lib/cn";
import { useCreateInquiry } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useDraftState } from "@/src/shared/lib/drafts";
import type { InstantResponse } from "@/src/shared/api/punisher";
import { ResolutionChoices } from "./ResolutionChoices";
import { inquiryHref, useLaunchInstant } from "./useLaunchInstant";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

type QueryMode = "answer" | "study";

type ModeText = "Label" | "Hint" | "Shortcut" | "Placeholder";
type ModeKey = `${QueryMode}${ModeText}`;

const MODES: { value: QueryMode; max: number; icon: typeof BookOpen }[] = [
  { value: "answer", max: 1000, icon: MessageCircleQuestion },
  { value: "study", max: 500, icon: BookOpen },
];

/**
 * 학습 홈 '궁금한 것' 입력 — 기본은 '이 질문에 답하기'(질문 자체를 해설), '이 개념 공부하기'는 기존 흐름
 * (개념을 찾아 일반 강의, design §3.2 B). 답하기는 긴 상황 설명을 쓰므로 Enter 가 줄바꿈이고(실수로 1~3분짜리
 * 생성을 시작하지 않게) ⌘/Ctrl+Enter·화살표 버튼으로 보낸다. 공부하기는 짧은 말이라 Enter 로 보낸다.
 */
export function InstantQueryBox() {
  const router = useRouter();
  const t = useT(messages);
  const text = (m: QueryMode, part: ModeText) => t[`${m}${part}` as ModeKey];
  const [mode, setMode] = useState<QueryMode>("answer");
  // 쓰던 질문은 이 기기에 보관한다 — 다른 화면에 갔다 와도 남고, 보내면 지운다(점검 17).
  const queryDraft = useDraftState("instant-query", "");
  const query = queryDraft.value;
  const setQuery = queryDraft.setValue;
  const [submitted, setSubmitted] = useState("");
  const [result, setResult] = useState<InstantResponse | null>(null);
  // 해설 화면으로 넘어가는 동안 한 번 더 보내지 않게(mutation 이 끝난 뒤에도 이동 전까지 막는다).
  const [navigating, setNavigating] = useState(false);
  const instant = useLaunchInstant();
  const inquiry = useCreateInquiry();
  const current = MODES.find((m) => m.value === mode) ?? MODES[0];
  const isPending = instant.isPending || inquiry.isPending || navigating;
  const error = mode === "answer" ? inquiry.error : instant.error;
  // 모드를 바꿔도 쓴 글은 그대로 둔다 — 한도를 넘으면 경고하고 보내기만 막는다.
  const tooLong = query.trim().length > current.max;

  async function submit() {
    const trimmed = query.trim();
    if (!trimmed || isPending || trimmed.length > current.max) { return; }
    setResult(null);
    setSubmitted(trimmed);
    try {
      if (mode === "answer") {
        const created = await inquiry.mutateAsync(trimmed);
        queryDraft.clear();
        setNavigating(true);
        router.push(inquiryHref(created.id));
        return;
      }
      setResult(await instant.launch({ query: trimmed }));
      queryDraft.clear();
    } catch {
      // 오류는 아래 <InlineError error={error}> 가 표시한다.
      setResult(null);
    }
  }

  async function handlePick(conceptId: string) {
    try {
      setResult(await instant.launch({ conceptId, query: submitted }));
    } catch {
      // 선택지를 유지하고 오류만 표시한다.
      return;
    }
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    void submit();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key !== "Enter" || event.nativeEvent.isComposing) { return; }
    const send = mode === "answer" ? event.metaKey || event.ctrlKey : !event.shiftKey;
    if (send) {
      event.preventDefault();
      void submit();
    }
  }

  function pickMode(next: QueryMode) {
    setMode(next);
    setResult(null);
    inquiry.reset();
    instant.reset();
  }

  return (
    <div className="flex flex-col gap-2">
      <SegmentedControl aria-label={t.modeGroup} value={mode} onValueChange={(v) => pickMode(v as QueryMode)}>
        {MODES.map((m) => (
          <SegmentedControlItem key={m.value} value={m.value}>
            <m.icon className="size-4" aria-hidden="true" />
            {text(m.value, "Label")}
          </SegmentedControlItem>
        ))}
      </SegmentedControl>
      <form role="search" onSubmit={handleSubmit} className="relative">
        <TextField
          multiline
          value={query}
          onValueChange={({ value }) => setQuery(value)}
          placeholder={text(current.value, "Placeholder")}
          textareaProps={{
            "aria-label": text(current.value, "Label"),
            "aria-describedby": "instant-query-hint",
            maxLength: 1000,
            rows: 3,
            enterKeyHint: mode === "answer" ? "enter" : "send",
            onKeyDown: handleKeyDown,
            className: "resize-y pr-12",
          }}
        />
        <IconButton
          type="submit"
          size="sm"
          aria-label={isPending ? t.sending : text(current.value, "Label")}
          disabled={isPending || !query.trim() || tooLong}
          className="absolute bottom-2 right-2"
          icon={isPending ? <ProgressCircle size="24" tone="inherit" aria-hidden /> : <ArrowRight aria-hidden="true" />}
        />
      </form>
      <div className="flex items-start justify-between gap-3 text-xs text-fg-muted">
        <p id="instant-query-hint" className="break-keep">
          {text(current.value, "Hint")}
          {/* JS 미디어쿼리는 첫 렌더 뒤에 바뀌어 모바일에서 줄이 밀리므로 CSS 로 숨긴다. */}
          <span className="[@media(pointer:coarse)]:hidden"> {text(current.value, "Shortcut")}</span>
        </p>
        <span role={tooLong ? "alert" : undefined} aria-hidden={!tooLong} className={cn("shrink-0 tabular-nums", tooLong && "font-medium text-danger")}>
          {tooLong && t.tooLong(current.max)}
          {query.length}/{current.max}
        </span>
      </div>
      <div aria-live="polite" className="flex flex-col gap-2 empty:hidden">
        <InlineError error={error} />
        {mode === "study" && result && <ResolutionChoices response={result} disabled={isPending} onPick={handlePick} />}
      </div>
    </div>
  );
}
