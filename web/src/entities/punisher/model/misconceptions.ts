
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { punisherApi, punisherKeys, punisherQueryDefaults, type CheckDetail, type MisconceptionStatus } from "@/src/shared/api/punisher";

/** 인정한 오개념 목록(open = 아직, resolved = 고친 것, 최근순). */
export function useMisconceptions(status: MisconceptionStatus) {
  return useQuery({
    queryKey: punisherKeys.misconceptions(status),
    queryFn: () => punisherApi.listMisconceptions(status),
    ...punisherQueryDefaults,
  });
}

/** 세션 캐시의 인정한 턴 목록에 turnId 를 더한다(이미 있으면 그대로). */
export function withAcknowledged(detail: CheckDetail, turnId: string): CheckDetail {
  const ids = detail.misconceptionTurnIds ?? [];
  return ids.includes(turnId) ? detail : { ...detail, misconceptionTurnIds: [...ids, turnId] };
}

/** '맞아요, 그렇게 알았어요' — 서버가 그 턴의 오개념을 저장한다(같은 턴은 한 번). */
export function useAcknowledgeMisconception(checkId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (turnId: string) => punisherApi.acknowledgeMisconception(turnId),
    onSuccess: (m) => {
      queryClient.setQueryData<CheckDetail>(punisherKeys.check(checkId), (prev) => (prev ? withAcknowledged(prev, m.turnId) : prev));
      void queryClient.invalidateQueries({ queryKey: ["punisher", "misconceptions"] });
    },
  });
}
