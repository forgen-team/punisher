
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type FeedbackKind,
  type InstantRequest,
  type Question,
} from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

export function useToday(limit = 5) {
  return useQuery({
    queryKey: punisherKeys.today(),
    queryFn: () => punisherApi.getToday(limit),
    ...punisherQueryDefaults,
  });
}

export function useStudyRequests(limit = 20) {
  return useQuery({
    queryKey: punisherKeys.studyRequests(),
    queryFn: () => punisherApi.listStudyRequests(limit),
    ...punisherQueryDefaults,
  });
}

export function useStartInstant() {
  return useMutation({ mutationFn: (body: InstantRequest) => punisherApi.startInstant(body) });
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
