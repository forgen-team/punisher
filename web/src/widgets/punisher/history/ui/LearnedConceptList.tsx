
import { useMemo } from "react";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@forgen-team/ui";
import { ConceptStateBadge, areaLabel, useConcepts } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 완료·이미 앎 개념 (최근 갱신순). 행당 상태 배지 1개. */
export function LearnedConceptList() {
  const t = useT(messages);
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
    return <ErrorState title={describePunisherError(concepts.error)} onRetry={() => concepts.refetch()} />;
  }
  if (learned.length === 0) {
    return <EmptyState title={t.noLearned} description={t.noLearnedDescription} />;
  }
  // 넓은 화면(xl)은 두 열 — 짧은 행이 화면 끝까지 늘어나지 않게.
  return (
    <List className="xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>li:nth-last-child(2):nth-child(odd)]:border-b-0">
      {learned.map((c) => (
        <ListRow
          key={c.id}
          href={`/map?concept=${encodeURIComponent(c.id)}`}
          title={c.name}
          meta={t.areaDepth(areaLabel(c.area), c.depth)}
          trailing={<ConceptStateBadge state={c.userState} />}
        />
      ))}
    </List>
  );
}
