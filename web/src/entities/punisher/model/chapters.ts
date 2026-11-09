
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { punisherApi, punisherKeys, punisherQueryDefaults } from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

/** "다음엔 어떤 걸 공부해 보고 싶어요?" 제안(3~5개). */
export function useChapterSuggestions() {
  return useQuery({
    queryKey: punisherKeys.chapterSuggestions(),
    queryFn: () => punisherApi.chapterSuggestions(),
    ...punisherQueryDefaults,
  });
}

/** 고른 제안·직접 입력으로 챕터 초안(미리보기)을 만든다 — 지도는 아직 바뀌지 않는다. */
export function useCreateChapterDraft() {
  return useMutation({
    mutationFn: (body: { suggestionIds: string[]; freeText: string }) => punisherApi.createChapterDraft(body),
  });
}

/** '이대로 열기' — 초안을 내 지도에 반영한다. */
export function useApplyChapterDraft() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (draftId: string) => punisherApi.applyChapterDraft(draftId),
    onSuccess: () => {
      void invalidateLearningState(queryClient);
      void queryClient.invalidateQueries({ queryKey: punisherKeys.chapterSuggestions() });
    },
  });
}
