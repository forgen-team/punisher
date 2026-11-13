import type { LinkedLecture } from "@/src/shared/api/punisher";

/** 다시 만든 버전 중 공개된 가장 최근 것(없으면 undefined) — 검수 배너가 또 만들지 않고 이 강의로 보낸다. */
export function latestPublished(versions: LinkedLecture[]): LinkedLecture | undefined {
  return versions
    .filter((l) => l.status === "published")
    .reduce<LinkedLecture | undefined>((best, l) => (!best || l.linkedAt > best.linkedAt ? l : best), undefined);
}
