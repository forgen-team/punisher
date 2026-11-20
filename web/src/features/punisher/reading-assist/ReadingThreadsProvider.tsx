import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from "react";
import { failedQuestionOf, useCreateNote, useQuestions, useStartDeepLecture, useStartThread } from "@/src/entities/punisher";
import type { InstantResponse, ThreadMode, ThreadStartInput } from "@/src/shared/api/punisher";
import type { OverlayAnchor } from "@/src/shared/ui/punisher";
import { groupPassages, groupThreads, type PassageThreads, type ReadingThread } from "./lib/threads";

/** 고른 구절(드래그 선택·탭한 문장·용어). */
export interface ThreadPassage {
  anchor: string;
  quote: string;
  /** 데스크톱 팝오버를 붙일 자리(열 때의 뷰포트 좌표) */
  rect: OverlayAnchor | null;
}

/** 지금 열린 스레드 — 아직 루트가 없으면 draft(질문 입력 대기·답 쓰는 중·실패). */
export type ThreadFocus =
  | { kind: "draft"; passage: ThreadPassage; mode: ThreadMode }
  | { kind: "thread"; rootId: string; rect: OverlayAnchor | null };

interface ReadingThreadsValue {
  lectureId: string;
  conceptId: string;
  getAnchor: () => string;
  threads: ReadingThread[];
  passages: PassageThreads[];
  focus: ThreadFocus | null;
  /** 새 스레드를 열 때마다(start·open) 1씩 — 답이 와서 draft 가 스레드로 바뀌는 것은 세지 않는다(패널 탭 전환용). */
  openSeq: number;
  /** 쉽게·깊게는 바로 답을 요청하고, 질문은 입력을 연다. */
  start: (passage: ThreadPassage, mode: ThreadMode) => void;
  /** draft(질문)에서 질문 보내기 */
  submitQuestion: (question: string) => void;
  /** draft 실패 다시 시도 */
  retryDraft: () => void;
  draftPending: boolean;
  draftError: unknown;
  open: (rootId: string, rect?: OverlayAnchor | null) => void;
  close: () => void;
  saveMemo: (passage: ThreadPassage) => Promise<void>;
  deep: {
    start: (rootId: string) => void;
    pending: boolean;
    error: unknown;
    /** 시작 응답(강의 위 상태 줄이 쓴다) */
    result: InstantResponse | null;
    /** 심화 강의를 만든 스레드의 루트 id */
    rootId: string;
    /** 지금(또는 마지막으로) 요청한 스레드의 루트 id — pending·error 는 이 스레드 것이다 */
    target: string;
    clear: () => void;
  };
}

const ReadingThreadsContext = createContext<ReadingThreadsValue | null>(null);

/** 강의 화면 밖(용어 목록 단독 사용 등)에서는 null — 스레드 버튼을 숨긴다. */
export function useReadingThreads(): ReadingThreadsValue | null {
  return useContext(ReadingThreadsContext);
}

interface ReadingThreadsProviderProps {
  lectureId: string;
  conceptId: string;
  /** 앵커 없이 시작할 때(용어 목록 등) 쓰는 읽던 위치 */
  getAnchor: () => string;
  children: ReactNode;
}

/**
 * 읽기 스레드 상태(④) — 본문 도구 막대·문장 탭·용어 시트가 스레드를 시작하고,
 * 모바일 시트·데스크톱 오른쪽 패널·본문 '대화 n' 표시가 같은 목록(GET 질문 목록을 스레드로 묶은 것)을 읽는다.
 */
export function ReadingThreadsProvider({ lectureId, conceptId, getAnchor, children }: ReadingThreadsProviderProps) {
  const questions = useQuestions(lectureId);
  const create = useStartThread(lectureId);
  const note = useCreateNote();
  const deepMutation = useStartDeepLecture(lectureId);
  const [focus, setFocus] = useState<ThreadFocus | null>(null);
  const [openSeq, setOpenSeq] = useState(0);
  const [deepResult, setDeepResult] = useState<{ rootId: string; res: InstantResponse } | null>(null);
  // 답을 기다리는 동안 닫거나 다른 구절을 고르면, 늦게 온 답이 시트를 다시 열지 않게 한다.
  const draftToken = useRef(0);
  const lastInput = useRef<{ input: ThreadStartInput; rect: OverlayAnchor | null } | null>(null);

  const threads = useMemo(() => groupThreads(questions.data ?? []), [questions.data]);
  const passages = useMemo(() => groupPassages(threads), [threads]);

  const send = useCallback(
    (input: ThreadStartInput, rect: OverlayAnchor | null) => {
      const token = ++draftToken.current;
      lastInput.current = { input, rect };
      create.reset();
      create.mutate(input, {
        onSuccess: (question) => {
          if (token === draftToken.current) { setFocus({ kind: "thread", rootId: question.id, rect }); }
        },
        // 502 는 저장된 실패 루트를 함께 보낸다 — 그 스레드를 열어, 다시 시도가 새 루트 대신 그 루트에 이어 묻게 한다.
        onError: (error) => {
          const failed = failedQuestionOf(error);
          if (failed && token === draftToken.current) { setFocus({ kind: "thread", rootId: failed.id, rect }); }
        },
      });
    },
    [create],
  );

  const start = useCallback(
    (passage: ThreadPassage, mode: ThreadMode) => {
      setFocus({ kind: "draft", passage, mode });
      setOpenSeq((n) => n + 1);
      if (mode === "question") {
        draftToken.current += 1;
        create.reset();
        return;
      }
      send({ anchor: passage.anchor || getAnchor(), quote: passage.quote, mode }, passage.rect);
    },
    [create, send, getAnchor],
  );

  const submitQuestion = useCallback(
    (question: string) => {
      if (focus?.kind !== "draft") { return; }
      const { passage } = focus;
      send({ anchor: passage.anchor || getAnchor(), quote: passage.quote, mode: "question", question }, passage.rect);
    },
    [focus, send, getAnchor],
  );

  const retryDraft = useCallback(() => {
    if (lastInput.current) { send(lastInput.current.input, lastInput.current.rect); }
  }, [send]);

  const open = useCallback((rootId: string, rect: OverlayAnchor | null = null) => {
    draftToken.current += 1;
    setFocus({ kind: "thread", rootId, rect });
    setOpenSeq((n) => n + 1);
  }, []);

  const close = useCallback(() => {
    draftToken.current += 1;
    setFocus(null);
  }, []);

  const saveMemo = useCallback(
    async (passage: ThreadPassage) => {
      await note.mutateAsync({ lectureId, conceptId, anchor: passage.anchor || getAnchor(), kind: "highlight", quote: passage.quote, body: "" });
    },
    [note, lectureId, conceptId, getAnchor],
  );

  const deepStart = useCallback(
    (rootId: string) => {
      deepMutation.mutate(rootId, { onSuccess: (res) => setDeepResult({ rootId, res }) });
    },
    [deepMutation],
  );

  const value = useMemo<ReadingThreadsValue>(
    () => ({
      lectureId,
      conceptId,
      getAnchor,
      threads,
      passages,
      focus,
      openSeq,
      start,
      submitQuestion,
      retryDraft,
      draftPending: create.isPending,
      draftError: create.error,
      open,
      close,
      saveMemo,
      deep: {
        start: deepStart,
        pending: deepMutation.isPending,
        error: deepMutation.error,
        result: deepResult?.res ?? null,
        rootId: deepResult?.rootId ?? "",
        target: deepMutation.variables ?? "",
        clear: () => setDeepResult(null),
      },
    }),
    [lectureId, conceptId, getAnchor, threads, passages, focus, openSeq, start, submitQuestion, retryDraft, create.isPending, create.error, open, close, saveMemo, deepStart, deepMutation.isPending, deepMutation.error, deepMutation.variables, deepResult],
  );

  return <ReadingThreadsContext.Provider value={value}>{children}</ReadingThreadsContext.Provider>;
}
