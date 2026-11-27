
import { EmptyState, ErrorState, List, ListRow, ListSkeleton, Section } from "@forgen-team/ui";
import { describeCheckError, useMisconceptions } from "@/src/entities/punisher";
import { resolvedMeta } from "../lib/checkHistory";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 고친 오개념 — 인정한 오개념 중 같은 목표를 혼자 맞혀 해소된 것(최근순). */
export function ResolvedMisconceptions() {
  const locale = useLocale();
  const t = useT(messages);
  const list = useMisconceptions("resolved");
  return <Section title={t.resolved}>{renderBody()}</Section>;

  function renderBody() {
    if (list.isLoading) { return <ListSkeleton rows={3} />; }
    if (list.error || !list.data) {
      return <ErrorState title={describeCheckError(list.error)} onRetry={() => list.refetch()} />;
    }
    if (list.data.length === 0) { return <EmptyState title={t.noResolved} />; }
    return (
      <List>
        {list.data.map((m) => (
          <ListRow key={m.id} title={m.text} meta={resolvedMeta(m, locale)} />
        ))}
      </List>
    );
  }
}
