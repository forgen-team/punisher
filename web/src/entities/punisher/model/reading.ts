
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type LearningSettingsPatch,
} from "@/src/shared/api/punisher";
import { invalidateLearningState } from "./concepts";

/** 이어보기에서 제거(소프트) — 오늘·내 강의·지도 상태를 다시 읽는다. */
export function useDismissLecture() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (lectureId: string) => punisherApi.dismissLecture(lectureId),
    onSettled: () => invalidateLearningState(queryClient),
  });
}

/** 이어보기 제거 되돌리기. */
export function useRestoreLecture() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (lectureId: string) => punisherApi.restoreLecture(lectureId),
    onSettled: () => invalidateLearningState(queryClient),
  });
}

export function useLearningSettings() {
  return useQuery({
    queryKey: punisherKeys.settings(),
    queryFn: () => punisherApi.getLearningSettings(),
    ...punisherQueryDefaults,
  });
}

export function useSaveLearningSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (patch: LearningSettingsPatch) => punisherApi.saveLearningSettings(patch),
    onSuccess: (data, patch) => {
      queryClient.setQueryData(punisherKeys.settings(), data);
      // 숙달 기준·복습 설정은 숙달 링·오늘 복습 카드·복습 요약에 바로 반영한다.
      if (patch.masteryTarget !== undefined) { void queryClient.invalidateQueries({ queryKey: ["punisher", "mastery"] }); }
      if (patch.reviewMinutes !== undefined || patch.reviewGrader !== undefined || patch.reviewIntervals !== undefined) {
        void queryClient.invalidateQueries({ queryKey: punisherKeys.today() });
        void queryClient.invalidateQueries({ queryKey: punisherKeys.reviewSummary() });
      }
    },
  });
}

/** "이 부분 설명" — 선택 구절을 섹션 맥락으로 설명(저장하지 않음). */
export function useExplainSelection(lectureId: string) {
  return useMutation({
    mutationFn: ({ anchor, selection }: { anchor: string; selection: string }) =>
      punisherApi.explainSelection(lectureId, anchor, selection),
  });
}

/** "이 개념 더 깊게"로 이어진 원 강의(parents)·심화 강의(children). */
export function useLectureLinks(lectureId: string) {
  return useQuery({
    queryKey: punisherKeys.lectureLinks(lectureId),
    queryFn: () => punisherApi.getLectureLinks(lectureId),
    enabled: Boolean(lectureId),
    ...punisherQueryDefaults,
  });
}
