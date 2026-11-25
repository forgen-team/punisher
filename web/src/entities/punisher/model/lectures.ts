
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type FeedbackKind,
  type InstantRequest,
  type Question,
  type StudyRequest,
} from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

export function useToday(limit = 5) {
  return useQuery({
    queryKey: punisherKeys.today(),
    queryFn: () => punisherApi.getToday(limit),
    ...punisherQueryDefaults,
  });
}

/** 개념 찾기가 진행 중이거나 곧 시작될(시도 3회 미만) 요청이 있으면 3초마다 새로 고친다. */
const isPendingResolution = (r: StudyRequest) => r.status === "unresolved" && (r.resolving || r.resolveAttempts === 0);

export function useStudyRequests(limit = 20) {
  return useQuery({
    queryKey: punisherKeys.studyRequests(),
    queryFn: () => punisherApi.listStudyRequests(limit),
    ...punisherQueryDefaults,
    refetchInterval: (query) => ((query.state.data ?? []).some((r) => isPendingResolution(r)) ? 3000 : false),
  });
}

/** 단독판: 학습 요청 직접 추가. 서버가 바로 개념 찾기를 시작하므로 목록을 다시 불러온다. */
export function useCreateStudyRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (query: string) => punisherApi.createStudyRequest(query),
    onSettled: () => qc.invalidateQueries({ queryKey: punisherKeys.studyRequests() }),
  });
}

export function useRetryStudyRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => punisherApi.resolveStudyRequest(id),
    onSettled: () => qc.invalidateQueries({ queryKey: punisherKeys.studyRequests() }),
  });
}

/** 학습 요청 '목록에서 빼기'(dismissed) / 되돌리기 — 목록에는 바로 반영하고 실패하면 다시 불러온다. */
export function useDismissStudyRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, dismissed }: { id: string; dismissed: boolean }) =>
      dismissed ? punisherApi.dismissStudyRequest(id) : punisherApi.restoreStudyRequest(id),
    onMutate: ({ id, dismissed }) => {
      qc.setQueriesData<StudyRequest[]>({ queryKey: punisherKeys.studyRequests() }, (prev) =>
        prev?.map((r) => (r.id === id ? { ...r, status: dismissed ? "dismissed" : r.resolvedConceptIds?.length ? "resolved" : "unresolved" } : r)),
      );
    },
    onSettled: () => qc.invalidateQueries({ queryKey: punisherKeys.studyRequests() }),
  });
}

export function useStartInstant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: InstantRequest) => punisherApi.startInstant(body),
    // "더 깊게"로 연 강의는 원 강의 아래 목록에 바로 보여야 한다.
    onSuccess: (_res, body) => {
      if (body.parentLectureId) {
        void queryClient.invalidateQueries({ queryKey: punisherKeys.lectureLinks(body.parentLectureId) });
      }
    },
  });
}

export function useLecture(id: string) {
  return useQuery({
    queryKey: punisherKeys.lecture(id),
    queryFn: () => punisherApi.getLecture(id),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
    // Progress is saved while reading; re-fetching would jump the resume anchor.
    staleTime: Infinity,
  });
}

export function useSaveProgress(lectureId: string) {
  return useMutation({
    mutationFn: ({ anchor, percent }: { anchor: string; percent: number }) =>
      punisherApi.saveProgress(lectureId, anchor, percent),
  });
}

export function useCompleteLecture(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => punisherApi.completeLecture(lectureId),
    onSuccess: () => invalidateLearningState(queryClient),
  });
}

/** '완료' 되돌리기(되돌리기 알림). */
export function useUncompleteLecture(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => punisherApi.uncompleteLecture(lectureId),
    onSuccess: () => invalidateLearningState(queryClient),
  });
}

export function useLectureFeedback(lectureId: string) {
  return useMutation({
    mutationFn: ({ feedback, anchor }: { feedback: FeedbackKind; anchor: string }) =>
      punisherApi.sendFeedback(lectureId, feedback, anchor),
  });
}

export function useQuestions(lectureId: string) {
  return useQuery({
    queryKey: punisherKeys.questions(lectureId),
    queryFn: () => punisherApi.listQuestions(lectureId),
    enabled: Boolean(lectureId),
    ...punisherQueryDefaults,
    // 스레드 답이 백그라운드로 쓰이는 중(pending)이면 다 쓸 때까지 다시 읽는다.
    refetchInterval: (query) => ((query.state.data ?? []).some((q) => q.status === "pending" && q.mode) ? 3000 : false),
  });
}

/** Polls a question (alternative explanation) until it is answered or failed. */
export function useQuestionPoll(id: string | undefined) {
  return useQuery({
    queryKey: punisherKeys.question(id ?? ""),
    queryFn: () => punisherApi.getQuestion(id ?? ""),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
    refetchInterval: (query) => (query.state.data?.status === "pending" ? 3000 : false),
  });
}

export function useAskQuestion(lectureId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ anchor, question }: { anchor: string; question: string }) =>
      punisherApi.ask(lectureId, anchor, question),
    onSuccess: (question) => {
      queryClient.setQueryData<Question[]>(punisherKeys.questions(lectureId), (prev) =>
        prev ? [...prev.filter((q) => q.id !== question.id), question] : [question],
      );
    },
    onError: () => queryClient.invalidateQueries({ queryKey: punisherKeys.questions(lectureId) }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["punisher", "my-questions"] }),
  });
}
