
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherLearningStateKeys,
  punisherQueryDefaults,
} from "@/src/shared/api/punisher";

/** Invalidate every query that depends on learning state (today, graph, ...). */
export function invalidateLearningState(queryClient: QueryClient): Promise<void> {
  return Promise.all(
    punisherLearningStateKeys.map((queryKey) => queryClient.invalidateQueries({ queryKey: [...queryKey] })),
  ).then(() => undefined);
}

export function useConcepts(area?: string) {
  return useQuery({
    queryKey: punisherKeys.concepts(area),
    queryFn: () => punisherApi.listConcepts(area),
    ...punisherQueryDefaults,
    staleTime: 5 * 60_000,
  });
}

export function useConceptDetail(id: string | undefined) {
  return useQuery({
    queryKey: punisherKeys.concept(id ?? ""),
    queryFn: () => punisherApi.getConcept(id ?? ""),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
  });
}

export function useGraph(area?: string) {
  return useQuery({
    queryKey: punisherKeys.graph(area),
    queryFn: () => punisherApi.getGraph(area),
    ...punisherQueryDefaults,
  });
}

export function useFrontier(limit = 10) {
  return useQuery({
    queryKey: punisherKeys.frontier(),
    queryFn: () => punisherApi.getFrontier(limit),
    ...punisherQueryDefaults,
  });
}

export function useGloss(conceptId: string, enabled: boolean) {
  return useQuery({
    queryKey: punisherKeys.gloss(conceptId),
    queryFn: () => punisherApi.getGloss(conceptId),
    enabled: enabled && Boolean(conceptId),
    ...punisherQueryDefaults,
    staleTime: 30 * 60_000,
  });
}

export function useMarkKnown() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (conceptId: string) => punisherApi.markKnown(conceptId),
    onSuccess: () => invalidateLearningState(queryClient),
  });
}

/** '필요 없음' 토글: dismissed=true 면 빼고, false 면 되돌린다. 학습 상태 쿼리를 모두 새로 읽는다. */
export function useConceptDismiss() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ conceptId, dismissed }: { conceptId: string; dismissed: boolean }) =>
      dismissed ? punisherApi.dismissConcept(conceptId) : punisherApi.restoreConcept(conceptId),
    onSuccess: () => invalidateLearningState(queryClient),
  });
}
