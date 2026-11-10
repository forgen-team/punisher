import { useRouter } from "@/src/shared/lib/router";
import { useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import { ArrowRight, BookOpen, MessageCircleQuestion } from "lucide-react";
import { Chip, IconButton, Spinner, Textarea } from "@/src/shared/ui/kit";
import { useCreateInquiry } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import type { InstantResponse } from "@/src/shared/api/punisher";
import { ResolutionChoices } from "./ResolutionChoices";
import { inquiryHref, useLaunchInstant } from "./useLaunchInstant";

type QueryMode = "answer" | "study";

const MODES: { value: QueryMode; label: string; hint: string; placeholder: string; max: number; icon: typeof BookOpen }[] = [
  {
    value: "answer",
    label: "이 질문에 답하기",
    hint: "겪은 상황을 그대로 적으면 원인·확인 방법·대처까지 해설을 써 드려요(1~3분). Enter는 줄바꿈, 보내기는 화살표나 ⌘/Ctrl+Enter.",
    placeholder: "예) 서버는 살아 있는데 API를 쏘면 CORS 오류가 나요. 원인이 궁금해요",
    max: 1000,
    icon: MessageCircleQuestion,
  },
  {
    value: "study",
    label: "이 개념 공부하기",
    hint: "개념 이름이나 짧은 말로 지도에서 개념을 찾아 바로 강의를 열어요. Enter로 보내요.",
    placeholder: "예) Pod가 죽었다, 커넥션 풀",
    max: 500,
    icon: BookOpen,
  },
];

/**
 * 학습 홈 '궁금한 것' 입력 — 기본은 '이 질문에 답하기'(질문 자체를 해설), '이 개념 공부하기'는 기존 흐름
 * (개념을 찾아 일반 강의, design §3.2 B). 답하기는 긴 상황 설명을 쓰므로 Enter 가 줄바꿈이고(실수로 1~3분짜리
 * 생성을 시작하지 않게) ⌘/Ctrl+Enter·화살표 버튼으로 보낸다. 공부하기는 짧은 말이라 Enter 로 보낸다.
 */
export function InstantQueryBox() {
  const router = useRouter();
  const [mode, setMode] = useState<QueryMode>("answer");
  const [query, setQuery] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [result, setResult] = useState<InstantResponse | null>(null);
  // 해설 화면으로 넘어가는 동안 한 번 더 보내지 않게(mutation 이 끝난 뒤에도 이동 전까지 막는다).
  const [navigating, setNavigating] = useState(false);
  const radios = useRef<(HTMLButtonElement | null)[]>([]);
  const instant = useLaunchInstant();
  const inquiry = useCreateInquiry();
  const current = MODES.find((m) => m.value === mode) ?? MODES[0];
  const isPending = instant.isPending || inquiry.isPending || navigating;
  const error = mode === "answer" ? inquiry.error : instant.error;

  async function submit() {
    const trimmed = query.trim();
    if (!trimmed || isPending) { return; }
    setResult(null);
    setSubmitted(trimmed);
    try {
      if (mode === "answer") {
        const created = await inquiry.mutateAsync(trimmed);
        setNavigating(true);
        router.push(inquiryHref(created.id));
        return;
      }
      setResult(await instant.launch({ query: trimmed }));
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

  // radiogroup: 탭 정지는 선택된 것 하나, 방향키로 옮기며 고른다(WAI-ARIA radio 패턴).
  function handleRadioKey(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
    if (!step) { return; }
    event.preventDefault();
    const next = (index + step + MODES.length) % MODES.length;
    pickMode(MODES[next].value);
    radios.current[next]?.focus();
  }

  function pickMode(next: QueryMode) {
    setMode(next);
    // 공부하기는 500자까지라 긴 글은 줄여서 넘긴다(서버 400 방지).
    setQuery((text) => text.slice(0, (MODES.find((m) => m.value === next) ?? MODES[0]).max));
    setResult(null);
    inquiry.reset();
    instant.reset();
  }

  return (
    <div className="flex flex-col gap-2">
      <div role="radiogroup" aria-label="질문 방식" className="flex flex-wrap gap-2">
        {MODES.map((m, index) => (
          <Chip
            key={m.value}
            ref={(el) => {
              radios.current[index] = el;
            }}
            role="radio"
            aria-pressed={undefined}
            aria-checked={mode === m.value}
            tabIndex={mode === m.value ? 0 : -1}
            selected={mode === m.value}
            onClick={() => pickMode(m.value)}
            onKeyDown={(event) => handleRadioKey(event, index)}
            leading={<m.icon aria-hidden="true" />}
          >
            {m.label}
          </Chip>
        ))}
      </div>
      <form role="search" onSubmit={handleSubmit} className="relative">
        <Textarea
          aria-label={current.label}
          aria-describedby="instant-query-hint"
          value={query}
          maxLength={current.max}
          rows={3}
          enterKeyHint={mode === "answer" ? "enter" : "send"}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={current.placeholder}
          className="resize-y pr-12"
        />
        <IconButton
          type="submit"
          size="sm"
          aria-label={isPending ? "보내는 중" : current.label}
          disabled={isPending || !query.trim()}
          className="absolute bottom-2 right-2"
        >
          {isPending ? <Spinner size="sm" label={null} /> : <ArrowRight aria-hidden="true" />}
        </IconButton>
      </form>
      <div className="flex items-start justify-between gap-3 text-xs text-fg-muted">
        <p id="instant-query-hint" className="break-keep">
          {current.hint}
        </p>
        <span aria-hidden="true" className="shrink-0 tabular-nums">
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
