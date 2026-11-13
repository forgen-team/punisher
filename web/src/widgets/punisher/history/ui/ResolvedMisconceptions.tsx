
import { EmptyState, ErrorState, List, ListRow, ListSkeleton, Section } from "@/src/shared/ui/kit";
import { describeCheckError, useMisconceptions } from "@/src/entities/punisher";
import { resolvedMeta } from "../lib/checkHistory";

/** 고친 오개념 — 인정한 오개념 중 같은 목표를 혼자 맞혀 해소된 것(최근순). */
export function ResolvedMisconceptions() {
  const list = useMisconceptions("resolved");
  return <Section title="고친 오개념">{renderBody()}</Section>;

  function renderBody() {
    if (list.isLoading) { return <ListSkeleton rows={3} />; }
    if (list.error || !list.data) {
      return <ErrorState compact message={describeCheckError(list.error)} onRetry={() => list.refetch()} retrying={list.isRefetching} />;
    }
    if (list.data.length === 0) { return <EmptyState compact title="아직 고친 오개념이 없어요" />; }
    return (
      <List>
        {list.data.map((m) => (
          <ListRow key={m.id} title={m.text} meta={resolvedMeta(m)} />
        ))}
      </List>
    );
  }
}
