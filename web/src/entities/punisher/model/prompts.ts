
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { punisherApi, punisherKeys, punisherQueryDefaults, type PromptSaveResponse, type PromptsResponse } from "@/src/shared/api/punisher";

export function usePrompts() {
  return useQuery({ queryKey: punisherKeys.prompts(), queryFn: () => punisherApi.listPrompts(), ...punisherQueryDefaults });
}

export function usePromptHistory(id: string) {
  return useQuery({
    queryKey: punisherKeys.promptHistory(id),
    queryFn: () => punisherApi.promptHistory(id),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
  });
}

type PromptChange = { kind: "save"; id: string; body: string } | { kind: "reset"; id: string } | { kind: "restore"; id: string; historyId: number };

/** 저장·기본값으로 되돌리기·이전 버전 복원 — 목록 캐시를 바로 고치고 이력을 다시 읽는다. */
export function useChangePrompt() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (change: PromptChange): Promise<PromptSaveResponse> => {
      if (change.kind === "save") { return punisherApi.savePrompt(change.id, change.body); }
      if (change.kind === "reset") { return punisherApi.resetPrompt(change.id); }
      return punisherApi.restorePrompt(change.id, change.historyId);
    },
    onSuccess: (res, change) => {
      queryClient.setQueryData<PromptsResponse>(punisherKeys.prompts(), (prev) =>
        prev ? { ...prev, blocks: prev.blocks.map((b) => (b.id === res.block.id ? { ...b, ...res.block } : b)) } : prev,
      );
      void queryClient.invalidateQueries({ queryKey: punisherKeys.promptHistory(change.id) });
    },
  });
}

/** 미리보기(조립만, LLM 호출 없음). body 가 있으면 저장 전 초안으로 조립한다. */
export function usePromptPreview() {
  return useMutation({ mutationFn: ({ id, body }: { id: string; body?: string }) => punisherApi.previewPrompt(id, body) });
}

export function useRegenerateLecture() {
  return useMutation({ mutationFn: (lectureId: string) => punisherApi.regenerateLecture(lectureId) });
}
