
import { useQuery } from "@tanstack/react-query";
import { useLaunchInstant } from "@/src/features/punisher/instant-start";
import { punisherApi, punisherKeys, punisherQueryDefaults } from "@/src/shared/api/punisher";

/** 실패한 즉석 생성을 같은 개념·깊이로 다시 시작한다(새 잡 → 새 스트림 화면으로 이동). */
export function useRetryInstant(jobId: string) {
  const job = useQuery({ queryKey: punisherKeys.job(jobId), queryFn: () => punisherApi.getJob(jobId), ...punisherQueryDefaults });
  const { launch, isPending } = useLaunchInstant();
  const conceptId = job.data?.conceptId ?? "";
  return {
    available: Boolean(conceptId),
    isPending,
    run: () => {
      if (!conceptId) { return; }
      void launch({ conceptId, depth: job.data?.depth || undefined }).catch((err: unknown) => {
        console.warn("punisher: retry instant failed", err);
      });
    },
  };
}
