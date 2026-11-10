import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { punisherApi, punisherKeys, punisherQueryDefaults, type Inquiry } from "@/src/shared/api/punisher";

/** 쓰는 중인 해설은 이 간격으로 다시 확인한다(보통 1~3분). */
const PENDING_POLL_MS = 3000;

function pending(list: Inquiry[] | undefined): boolean {
  return Boolean(list?.some((q) => q.status === "pending"));
}

/** 내 질문 목록 — open: '다 봤어요' 하지 않은 것(학습 홈), concept: 그 개념에 붙은 것(지도). */
export function useInquiries(filter: { open?: boolean; concept?: string } = {}, enabled = true) {
  return useQuery({
    queryKey: punisherKeys.inquiries(filter),
    queryFn: () => punisherApi.listInquiries(filter),
    enabled,
    ...punisherQueryDefaults,
    refetchInterval: (query) => (pending(query.state.data) ? PENDING_POLL_MS : false),
  });
}

/** 질문 하나(해설 본문 포함) — 쓰는 중이면 끝날 때까지 다시 확인한다. 끝난 해설은 캐시 그대로. */
export function useInquiry(id: string) {
  return useQuery({
    queryKey: punisherKeys.inquiry(id),
    queryFn: () => punisherApi.getInquiry(id),
    enabled: Boolean(id),
    ...punisherQueryDefaults,
    refetchInterval: (query) => {
      const data = query.state.data;
      return data?.status === "pending" || data?.followups?.some((f) => f.status === "pending") ? PENDING_POLL_MS : false;
    },
  });
}

function useInvalidateInquiries() {
  const queryClient = useQueryClient();
  return (updated?: Inquiry) => {
    if (updated) { queryClient.setQueryData(punisherKeys.inquiry(updated.id), (prev: Inquiry | undefined) => ({ ...prev, ...updated })); }
    void queryClient.invalidateQueries({ queryKey: ["punisher", "inquiries"] });
    void queryClient.invalidateQueries({ queryKey: ["punisher", "graph"] });
    void queryClient.invalidateQueries({ queryKey: ["punisher", "completed"] });
    void queryClient.invalidateQueries({ queryKey: ["punisher", "completed-areas"] });
  };
}

/** '이 질문에 답하기' — 질문을 저장하고 해설을 쓰기 시작한다(202, 상태 pending). */
export function useCreateInquiry() {
  const invalidate = useInvalidateInquiries();
  return useMutation({
    mutationFn: (question: string) => punisherApi.createInquiry(question),
    onSuccess: (q) => invalidate(q),
  });
}

/** '다시 만들기' — 같은 질문으로 새 해설(새 해설이 저장될 때까지 이전 해설을 보인다). */
export function useRegenerateInquiry() {
  const invalidate = useInvalidateInquiries();
  return useMutation({
    mutationFn: (id: string) => punisherApi.regenerateInquiry(id),
    onSuccess: (q) => invalidate(q),
  });
}

/** '다 봤어요'(학습 홈 목록에서 빼기) / 되돌리기. 지도·기록에는 남는다. */
export function useSetInquiryRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, read }: { id: string; read: boolean }) => punisherApi.setInquiryRead(id, read),
    onSuccess: (_void, { id }) => {
      void queryClient.invalidateQueries({ queryKey: punisherKeys.inquiry(id) });
      void queryClient.invalidateQueries({ queryKey: ["punisher", "inquiries"] });
    },
  });
}

/** 이어서 묻기 — 스레드에 후속 질문을 더하고 답을 쓰기 시작한다. */
export function useCreateFollowup(inquiryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: { question: string; quote?: string }) => punisherApi.createFollowup(inquiryId, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: punisherKeys.inquiry(inquiryId) });
      void queryClient.invalidateQueries({ queryKey: ["punisher", "inquiries"] });
    },
  });
}

/** 실패한 후속 답 다시 시도. */
export function useRetryFollowup(inquiryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (followupId: string) => punisherApi.retryFollowup(inquiryId, followupId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: punisherKeys.inquiry(inquiryId) }),
  });
}

/** 완료 목록의 분야별 개수(검색어 반영, 서버 집계). */
export function useCompletedAreas(q: string) {
  return useQuery({
    queryKey: punisherKeys.completedAreas(q),
    queryFn: () => punisherApi.completedAreas(q),
    ...punisherQueryDefaults,
  });
}

const COMPLETED_PAGE = 20;

/** 완료한 강의·질문 스레드(최근 순) — 분야·검색은 서버에서 거른다. 접힌 분야는 불러오지 않는다. */
export function useCompleted(filter: { area?: string; q?: string }, enabled = true) {
  return useInfiniteQuery({
    queryKey: punisherKeys.completed(filter),
    queryFn: ({ pageParam }) => punisherApi.listCompleted({ ...filter, limit: COMPLETED_PAGE, cursor: pageParam }),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
    enabled,
    ...punisherQueryDefaults,
  });
}
