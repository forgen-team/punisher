import { useRouter } from "@/src/shared/lib/router";
import { useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import { ArrowRight, BookOpen, MessageCircleQuestion } from "lucide-react";
import { Chip, IconButton, Spinner, Textarea, cn } from "@/src/shared/ui/kit";
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
  const radios = useRef<(HTMLButtonElement | null)[]>([]);
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
    setResult(null);
    inquiry.reset();
    instant.reset();
  }

  return (
    <div className="flex flex-col gap-2">
      <div role="radiogroup" aria-label={t.modeGroup} className="flex flex-wrap gap-2">
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
            {text(m.value, "Label")}
          </Chip>
        ))}
      </div>
      <form role="search" onSubmit={handleSubmit} className="relative">
        <Textarea
          aria-label={text(current.value, "Label")}
          aria-describedby="instant-query-hint"
          value={query}
          maxLength={1000}
          rows={3}
          enterKeyHint={mode === "answer" ? "enter" : "send"}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={text(current.value, "Placeholder")}
          className="resize-y pr-12"
        />
        <IconButton
          type="submit"
          size="sm"
          aria-label={isPending ? t.sending : text(current.value, "Label")}
          disabled={isPending || !query.trim() || tooLong}
          className="absolute bottom-2 right-2"
        >
          {isPending ? <Spinner size="sm" label={null} /> : <ArrowRight aria-hidden="true" />}
        </IconButton>
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
