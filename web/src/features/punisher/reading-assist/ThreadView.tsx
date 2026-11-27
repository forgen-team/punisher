import { useEffect, useRef, useState, type FormEvent } from "react";
import { Layers } from "lucide-react";
import { Button, Skeleton, TextField } from "@forgen-team/ui";
import { failedQuestionOf, useThreadFollowup } from "@/src/entities/punisher";
import { localizeServerText, type Question, type ThreadFollowupInput, type ThreadMode } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useT } from "@/src/shared/i18n";
import { useReadingThreads, type ThreadFocus } from "./ReadingThreadsProvider";
import { findThread, type ReadingThread } from "./lib/threads";
import { messages } from "./i18n";
import { AddToReview } from "@/src/features/punisher/card-add";

type T = typeof messages.ko;

/** 내 요청 — 쉽게·깊게는 서버가 만든 요청 문장 대신 짧은 표지로, 질문은 말풍선으로 보인다. */
interface Request {
  mode?: ThreadMode;
  question?: string;
}

function modeLabel(mode: ThreadMode | undefined, isReply: boolean, t: T): string | null {
  if (mode === "easier") { return isReply ? t.moreEasier : t.easier; }
  if (mode === "deeper") { return isReply ? t.moreDeeper : t.deeper; }
  return null;
}

function RequestLine({ request, isReply }: { request: Request; isReply: boolean }) {
  const t = useT(messages);
  const label = modeLabel(request.mode, isReply, t);
  if (label) {
    return (
      <p className="self-start text-xs font-medium text-accent-text">
        <span className="sr-only">{t.you}: </span>
        {label}
      </p>
    );
  }
  return (
    <p className="self-end rounded-lg bg-canvas-subtle px-3 py-1.5 text-sm text-fg-secondary">
      <span className="sr-only">{t.you}: </span>
      {request.question || (isReply ? t.followupLabel : t.question)}
    </p>
  );
}

function Answering() {
  const t = useT(messages);
  return (
    <div aria-busy="true" aria-label={t.answering}>
      <Skeleton lines={3} />
    </div>
  );
}

/** 한 번의 묻고 답하기 — 내 요청(작게) + 답(강의 설명과 같은 마크다운). */
function Turn({ q, isReply }: { q: Question; isReply: boolean }) {
  const t = useT(messages);
  return (
    <li className="flex flex-col gap-1.5">
      <RequestLine request={q} isReply={isReply} />
      {q.status === "answered" && <Markdown className="text-base leading-[1.7]">{q.answer}</Markdown>}
      {q.status === "pending" && <Answering />}
      {q.status === "failed" && <p className="text-sm text-danger">{localizeServerText(q.error, t.answerFailed)}</p>}
    </li>
  );
}

function Quote({ text }: { text: string }) {
  return <blockquote className="line-clamp-4 border-l-2 border-line-strong pl-3 text-sm text-fg-secondary">{text}</blockquote>;
}

function AskForm({ label, pending, onSend, autoFocus }: { label: string; pending: boolean; onSend: (text: string) => void; autoFocus?: boolean }) {
  const t = useT(messages);
  const [text, setText] = useState("");
  const inputRef = useRef<HTMLInputElement & HTMLTextAreaElement>(null);
  useEffect(() => {
    if (autoFocus) { inputRef.current?.focus({ preventScroll: true }); }
  }, [autoFocus]);
  function submit(event: FormEvent) {
    event.preventDefault();
    const value = text.trim();
    if (!value || pending) { return; }
    onSend(value);
    setText("");
  }
  return (
    <form onSubmit={submit} className="flex items-center gap-2">
      <TextField
        inputRef={inputRef}
        placeholder={label}
        value={text}
        onValueChange={({ value }) => setText(value)}
        inputProps={{ "aria-label": label, maxLength: 2000 }}
        className="min-w-0 flex-1"
      />
      <Button type="submit" size="md" variant="secondary" loading={pending} disabled={!text.trim()}>
        {t.send}
      </Button>
    </form>
  );
}

/** 루트가 생긴 스레드 — 대화 · 더 쉽게/더 깊게 · 이어 묻기 · 심화 강의 만들기. */
function ThreadBody({ thread, showQuote, autoFocusInput }: { thread: ReadingThread; showQuote: boolean; autoFocusInput?: boolean }) {
  const t = useT(messages);
  const ctx = useReadingThreads();
  const followup = useThreadFollowup(ctx?.lectureId ?? "");
  const [lastFollowup, setLastFollowup] = useState<ThreadFollowupInput | null>(null);
  const rootId = thread.root.id;
  const answered = thread.root.status === "answered" || thread.replies.some((r) => r.status === "answered");
  // 루트가 실패했고 아직 답이 없으면 — 새 루트를 만들지 않고 이 루트에 같은 요청을 이어 묻는다.
  const rootFailedOnly =
    thread.root.status === "failed" && !answered && !thread.replies.some((r) => r.status === "pending") && !followup.isPending && !followup.isError;

  function ask(input: ThreadFollowupInput) {
    setLastFollowup(input);
    followup.mutate({ rootId, ...input });
  }

  function retryRoot() {
    const { mode, question } = thread.root;
    ask(mode === "easier" || mode === "deeper" ? { mode } : { mode: "question", question });
  }

  // 심화 강의 상태는 그 강의를 만든(만드는 중인) 스레드에서만 보인다.
  const deepMine = ctx?.deep.target === rootId;
  const deepStarted = Boolean(ctx?.deep.result) && ctx?.deep.rootId === rootId;

  return (
    <div className="flex flex-col gap-3">
      {showQuote && thread.root.quote && <Quote text={thread.root.quote} />}
      <ol className="flex flex-col gap-4">
        <Turn q={thread.root} isReply={false} />
        {thread.replies.map((r) => (
          <Turn key={r.id} q={r} isReply />
        ))}
        {followup.isPending && lastFollowup && (
          <li className="flex flex-col gap-1.5">
            <RequestLine request={lastFollowup} isReply />
            <Answering />
          </li>
        )}
      </ol>
      {rootFailedOnly && (
        <Button size="sm" variant="secondary" className="self-start" onClick={retryRoot}>
          {t.retry}
        </Button>
      )}
      {followup.isError && (
        <div aria-live="polite" className="flex items-center gap-2">
          {/* 502 는 실패 항목이 대화에 이미 보인다 — 같은 실패를 두 번 쓰지 않는다. */}
          {!failedQuestionOf(followup.error) && <InlineError error={followup.error} />}
          {lastFollowup && (
            <Button size="sm" variant="secondary" onClick={() => ask(lastFollowup)}>
              {t.retry}
            </Button>
          )}
        </div>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" disabled={followup.isPending || !answered} onClick={() => ask({ mode: "easier" })}>
          {t.moreEasier}
        </Button>
        <Button variant="outline" size="sm" disabled={followup.isPending || !answered} onClick={() => ask({ mode: "deeper" })}>
          {t.moreDeeper}
        </Button>
        {/* 구절 대화 → 복습 카드(W5a): 답이 있는 대화만 */}
        {answered && <AddToReview kind="thread" sourceId={rootId} />}
      </div>
      <AskForm label={t.followupLabel} pending={followup.isPending} onSend={(question) => ask({ question, mode: "question" })} autoFocus={autoFocusInput} />
      {ctx && (
        <div className="flex flex-col gap-1">
          <Button
            size="sm"
            variant="ghost"
            className="self-start"
            loading={ctx.deep.pending && deepMine}
            disabled={!answered || deepStarted || ctx.deep.pending}
            onClick={() => ctx.deep.start(rootId)}
          >
            <Layers className="size-4" aria-hidden="true" />
            {t.deepLecture}
          </Button>
          <div aria-live="polite" className="empty:hidden">
            {deepStarted && <p className="text-xs text-fg-muted">{t.deepStarted}</p>}
            {deepMine && <InlineError error={ctx.deep.error} />}
          </div>
        </div>
      )}
    </div>
  );
}

/** 루트를 만드는 중(질문 입력 · 답 쓰는 중 · 실패). */
function DraftBody({ quote, mode }: { quote: string; mode: ThreadMode }) {
  const t = useT(messages);
  const ctx = useReadingThreads();
  if (!ctx) { return null; }
  const waitingForQuestion = mode === "question" && !ctx.draftPending && !ctx.draftError;
  return (
    <div className="flex flex-col gap-3">
      <Quote text={quote} />
      {waitingForQuestion ? (
        <AskForm label={t.askLabel} pending={false} onSend={ctx.submitQuestion} autoFocus />
      ) : ctx.draftPending ? (
        <>
          {mode !== "question" && <RequestLine request={{ mode }} isReply={false} />}
          <Answering />
        </>
      ) : (
        <div aria-live="polite" className="flex flex-col items-start gap-2">
          <InlineError error={ctx.draftError} />
          <Button size="sm" variant="secondary" onClick={ctx.retryDraft}>
            {t.retry}
          </Button>
        </div>
      )}
    </div>
  );
}

interface ThreadViewProps {
  focus: ThreadFocus;
  threads: ReadingThread[];
  /** 패널에서 구절 머리에 이미 인용을 보이면 false */
  showQuote?: boolean;
}

/** 스레드 한 개(모바일 시트·데스크톱 패널 공용). 목록에 아직 없으면(방금 만든 루트) 답 쓰는 중으로 본다. */
export function ThreadView({ focus, threads, showQuote = true }: ThreadViewProps) {
  if (focus.kind === "draft") { return <DraftBody quote={focus.passage.quote} mode={focus.mode} />; }
  const thread = findThread(threads, focus.rootId);
  if (!thread) { return <Answering />; }
  return <ThreadBody thread={thread} showQuote={showQuote} />;
}

export { ThreadBody };
