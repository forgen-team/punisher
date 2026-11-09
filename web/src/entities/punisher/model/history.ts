
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  punisherApi,
  punisherKeys,
  punisherQueryDefaults,
  type MyLectureStatus,
} from "@/src/shared/api/punisher";

const PAGE_SIZE = 20;

/** The caller's lectures (completed or in progress), newest first, cursor-paginated. */
export function useMyLectures(status: MyLectureStatus) {
  return useInfiniteQuery({
    queryKey: punisherKeys.myLectures(status),
    queryFn: ({ pageParam }) => punisherApi.listMyLectures({ status, limit: PAGE_SIZE, cursor: pageParam }),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
    ...punisherQueryDefaults,
  });
}

/** The caller's questions across lectures; `conceptId` narrows to one concept. */
export function useMyQuestions(conceptId?: string) {
  return useInfiniteQuery({
    queryKey: punisherKeys.myQuestions(conceptId),
    queryFn: ({ pageParam }) => punisherApi.listMyQuestions({ conceptId, limit: PAGE_SIZE, cursor: pageParam }),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
    ...punisherQueryDefaults,
  });
}

/** Published/instant lectures of a concept with the caller's progress. */
export function useConceptLectures(conceptId: string | undefined) {
  return useQuery({
    queryKey: punisherKeys.conceptLectures(conceptId ?? ""),
    queryFn: () => punisherApi.listConceptLectures(conceptId ?? ""),
    enabled: Boolean(conceptId),
    ...punisherQueryDefaults,
  });
}
