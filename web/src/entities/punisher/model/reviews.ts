
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  MASTERY_BATCH_MAX,
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type ConceptMasterySummary,
} from "@/src/shared/api/punisher";
import { invalidateMastery } from "./checks";

const HISTORY_PAGE_SIZE = 20;

/** 복습 요약(만기·밀림·대기 수, 오늘 세션). */
export function useReviewSummary(enabled = true) {
  return useQuery({
    queryKey: punisherKeys.reviewSummary(),
    queryFn: () => punisherApi.getReviewSummary(),
    enabled,
    ...punisherQueryDefaults,
  });
}

/** '복습 시작'·'오늘 더 하기' — 오늘 복습 세션을 받는다(열린 세션이 있으면 그것, 끝났으면 새로). */
export function useStartReview() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => punisherApi.startReview(),
    onSuccess: (detail) => {
      queryClient.setQueryData(punisherKeys.check(detail.session.id), detail);
      invalidateMastery(queryClient);
    },
  });
}

/** 확인·복습 세션 기록(최신순, 커서). */
export function useCheckHistory() {
  return useInfiniteQuery({
    queryKey: punisherKeys.checkHistory(),
    queryFn: ({ pageParam }) => punisherApi.listCheckHistory({ limit: HISTORY_PAGE_SIZE, cursor: pageParam }),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
    ...punisherQueryDefaults,
  });
}

/** 기억 유지 지표(간격 × 채점 주체, 최근 4주). */
export function useMasteryMetrics() {
  return useQuery({
    queryKey: punisherKeys.masteryMetrics(),
    queryFn: () => punisherApi.getMasteryMetrics(),
    ...punisherQueryDefaults,
  });
}

/** 개념 id 목록을 일괄 조회 한도로 나눈다. */
export function chunkIds(ids: string[], size = MASTERY_BATCH_MAX): string[][] {
  const out: string[][] = [];
  for (let index = 0; index < ids.length; index += size) { out.push(ids.slice(index, index + size)); }
  return out;
}

/** 여러 개념의 숙달 요약(지도 링) — 300개씩 나눠 한 번에 받는다. */
export function useMasteryBatch(conceptIds: string[]) {
  const sorted = [...new Set(conceptIds)].sort();
  const idsKey = sorted.join(",");
  return useQuery({
    queryKey: punisherKeys.masteryBatch(idsKey),
    queryFn: async (): Promise<ConceptMasterySummary[]> => {
      const pages = await Promise.all(chunkIds(sorted).map((ids) => punisherApi.getMasteryBatch(ids)));
      return pages.flat();
    },
    enabled: sorted.length > 0,
    ...punisherQueryDefaults,
  });
}
