
import { Link } from "@/src/shared/lib/router";
import { useMemo } from "react";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { ConceptStateBadge, areaLabel, useConcepts } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";

/** 완료·이미 앎 개념 (최근 갱신순). 행당 상태 배지 1개. */
export function LearnedConceptList() {
  const concepts = useConcepts();
  const learned = useMemo(
    () =>
      (concepts.data ?? [])
        .filter((c) => c.userState === "completed" || c.userState === "known")
        .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)),
    [concepts.data],
  );
  if (concepts.isLoading) { return <ListSkeleton rows={5} />; }
  if (concepts.error) {
    return <ErrorState message={describePunisherError(concepts.error)} onRetry={() => concepts.refetch()} />;
  }
  if (learned.length === 0) {
    return <EmptyState title="아직 완료하거나 이미 앎으로 표시한 개념이 없어요" description="지도에서 이미 아는 개념을 표시할 수 있어요." />;
  }
  // 넓은 화면(xl)은 두 열 — 짧은 행이 화면 끝까지 늘어나지 않게.
  return (
    <List className="xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>li:nth-last-child(2):nth-child(odd)]:border-b-0">
      {learned.map((c) => (
        <ListRow
          key={c.id}
          href={`/map?concept=${encodeURIComponent(c.id)}`}
          linkAs={Link}
          titleLines={1}
          title={c.name}
          meta={`${areaLabel(c.area)} · 깊이 ${c.depth}`}
          trailing={<ConceptStateBadge state={c.userState} />}
        />
      ))}
    </List>
  );
}
