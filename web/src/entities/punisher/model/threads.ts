import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  PunisherApiError,
  punisherApi,
  punisherKeys,
  type Question,
  type ThreadFollowupInput,
  type ThreadStartInput,
} from "@/src/shared/api/punisher";

/** 답이 온 질문을 목록 캐시에 바로 넣고(같은 id 는 교체), 서버 목록으로 다시 맞춘다. */
function upsertQuestion(queryClient: QueryClient, lectureId: string, question: Question) {
  queryClient.setQueryData<Question[]>(punisherKeys.questions(lectureId), (prev) =>
    prev ? [...prev.filter((q) => q.id !== question.id), question] : [question],
  );
}

/**
 * 스레드 시작·이어 묻기의 502 는 저장된 실패 항목({question}, status=failed)을 함께 보낸다 — 그 항목, 없으면 null.
 * 실패한 루트를 다시 시도할 때 새 루트를 만들지 않고 그 루트에 이어 묻게 하려고 쓴다.
 */
export function failedQuestionOf(error: unknown): Question | null {
  if (!(error instanceof PunisherApiError) || error.body === null || typeof error.body !== "object") { return null; }
  const question = (error.body as { question?: unknown }).question;
  if (question === null || typeof question !== "object") { return null; }
  const q = question as Partial<Question>;
  return typeof q.id === "string" && q.id && typeof q.status === "string" ? (q as Question) : null;
}

/** 실패 응답에 실린 실패 항목도 목록 캐시에 넣는다(서버 목록에는 이미 있다). */
function upsertFailed(queryClient: QueryClient, lectureId: string, error: unknown) {
  const failed = failedQuestionOf(error);
  if (failed) { upsertQuestion(queryClient, lectureId, failed); }
}

function settleThread(queryClient: QueryClient, lectureId: string, mode: ThreadStartInput["mode"] | undefined) {
  void queryClient.invalidateQueries({ queryKey: punisherKeys.questions(lectureId) });
  void queryClient.invalidateQueries({ queryKey: ["punisher", "my-questions"] });
  // '쉽게'가 쌓이면 설정 응답에 수준 보정 제안이 생길 수 있다.
  if (mode === "easier") { void queryClient.invalidateQueries({ queryKey: punisherKeys.settings() }); }
}

/** 구절 스레드 시작(쉽게·깊게·질문) — 루트 질문을 만들고 답한다(④). */
export function useStartThread(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ThreadStartInput) => punisherApi.startThread(lectureId, input),
    onSuccess: (question) => upsertQuestion(queryClient, lectureId, question),
    onError: (error) => upsertFailed(queryClient, lectureId, error),
    onSettled: (_q, _e, input) => settleThread(queryClient, lectureId, input.mode),
  });
}

/** 이어 묻기 — 항상 스레드의 루트 id 로 보낸다(이어 묻기의 이어 묻기도 같은 루트). */
export function useThreadFollowup(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ rootId, ...input }: ThreadFollowupInput & { rootId: string }) => punisherApi.followupThread(rootId, input),
    onSuccess: (question) => upsertQuestion(queryClient, lectureId, question),
    onError: (error) => upsertFailed(queryClient, lectureId, error),
    onSettled: (_q, _e, input) => settleThread(queryClient, lectureId, input.mode),
  });
}

/** 이 대화로 심화 강의 만들기 — 원 강의 아래 '이어진 강의'로 붙는다. */
export function useStartDeepLecture(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (rootId: string) => punisherApi.startDeepLecture(rootId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: punisherKeys.lectureLinks(lectureId) }),
  });
}

/** 수준 보정 제안 닫기(7일) — 설정을 다시 읽어 카드를 내린다. */
export function useDismissLevelSuggestion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => punisherApi.dismissLevelSuggestion(),
    onSettled: () => queryClient.invalidateQueries({ queryKey: punisherKeys.settings() }),
  });
}
