import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  punisherKeys,
  punisherQueryDefaults,
  studyApi,
  studyKeys,
  type CardRating,
  type CardSourceKind,
  type LeechAction,
  type StudyPrefsPatch,
} from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

/** 복습·카드·활동 숫자가 걸린 쿼리를 새로 읽는다. */
function invalidateStudy(queryClient: QueryClient) {
  for (const key of [studyKeys.activity(), studyKeys.forecast(), studyKeys.leeches(), ["punisher", "cards"], punisherKeys.today()]) {
    void queryClient.invalidateQueries({ queryKey: key });
  }
}

export function useStudyPrefs() {
  return useQuery({ queryKey: studyKeys.prefs(), queryFn: studyApi.getPrefs, ...punisherQueryDefaults });
}

export function useSaveStudyPrefs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (patch: StudyPrefsPatch) => studyApi.savePrefs(patch),
    onSuccess: (data) => {
      queryClient.setQueryData(studyKeys.prefs(), data);
      invalidateStudy(queryClient);
    },
  });
}

export function useLeeches() {
  return useQuery({ queryKey: studyKeys.leeches(), queryFn: studyApi.leeches, ...punisherQueryDefaults });
}

export function useLeechAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ objectiveId, action }: { objectiveId: string; action: LeechAction }) => studyApi.setLeech(objectiveId, action),
    onSuccess: () => {
      invalidateStudy(queryClient);
      void invalidateLearningState(queryClient);
    },
  });
}

export function useReviewForecast() {
  return useQuery({ queryKey: studyKeys.forecast(), queryFn: () => studyApi.forecast(14), ...punisherQueryDefaults });
}

export function useActivity() {
  return useQuery({ queryKey: studyKeys.activity(), queryFn: () => studyApi.activity(84), ...punisherQueryDefaults });
}

export function useGoalPlan(goalId: string | undefined) {
  return useQuery({
    queryKey: studyKeys.goalPlan(goalId ?? ""),
    queryFn: () => studyApi.goalPlan(goalId ?? ""),
    enabled: Boolean(goalId),
    ...punisherQueryDefaults,
  });
}

export function useGoalDeadlines() {
  return useQuery({ queryKey: studyKeys.deadlines(), queryFn: studyApi.deadlines, ...punisherQueryDefaults });
}

export function useSetGoalDeadline() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ goalId, dueOn }: { goalId: string; dueOn: string }) => studyApi.setDeadline(goalId, dueOn),
    onSuccess: (_, { goalId }) => {
      void queryClient.invalidateQueries({ queryKey: studyKeys.deadlines() });
      void queryClient.invalidateQueries({ queryKey: studyKeys.goalPlan(goalId) });
    },
  });
}

export function useCardDraft(kind: CardSourceKind, sourceId: string | null) {
  return useQuery({
    queryKey: studyKeys.cardDraft(kind, sourceId ?? ""),
    queryFn: () => studyApi.cardDraft(kind, sourceId ?? ""),
    enabled: Boolean(sourceId),
    ...punisherQueryDefaults,
    staleTime: 0,
  });
}

export function useSaveCard() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: studyApi.saveCard,
    onSuccess: (card) => {
      invalidateStudy(queryClient);
      void queryClient.invalidateQueries({ queryKey: studyKeys.cardDraft(card.sourceKind, card.sourceId) });
    },
  });
}

export function useDeleteCard() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => studyApi.deleteCard(id),
    onSuccess: () => {
      invalidateStudy(queryClient);
      void queryClient.invalidateQueries({ queryKey: ["punisher", "card-draft"] });
    },
  });
}

export function useCards(due: boolean) {
  return useQuery({ queryKey: studyKeys.cards(due), queryFn: () => studyApi.listCards(due), ...punisherQueryDefaults });
}

export function useGradeCard() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rating }: { id: string; rating: CardRating }) => studyApi.gradeCard(id, rating),
    onSuccess: () => {
      for (const key of [studyKeys.activity(), studyKeys.forecast(), punisherKeys.today()]) {
        void queryClient.invalidateQueries({ queryKey: key });
      }
    },
  });
}

export function useSearch(q: string) {
  const trimmed = q.trim();
  return useQuery({
    queryKey: studyKeys.search(trimmed),
    queryFn: () => studyApi.search(trimmed),
    enabled: trimmed.length > 0,
    ...punisherQueryDefaults,
    staleTime: 60_000,
  });
}
