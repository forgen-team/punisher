
import { Link } from "@/src/shared/lib/router";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { describeCheckError, useCheckHistory } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { historyMeta, historyTitle } from "../lib/checkHistory";
import { LoadMore } from "./LoadMore";
import { RetentionMetrics } from "./RetentionMetrics";

/** 확인·복습 세션(최신순) + 아래 기억 유지 지표. 누르면 그 세션 화면(check/[id]). */
export function CheckHistoryList() {
  const history = useCheckHistory();
  const sessions = history.data?.pages.flatMap((p) => p.sessions) ?? [];
  return (
    <div className="flex flex-col gap-8">
      {renderSessions()}
      <RetentionMetrics />
    </div>
  );

  function renderSessions() {
    if (history.isLoading) { return <ListSkeleton rows={5} />; }
    if (history.error) {
      return <ErrorState message={describeCheckError(history.error)} onRetry={() => history.refetch()} retrying={history.isRefetching} />;
    }
    if (sessions.length === 0) {
      return <EmptyState title="아직 확인·복습 기록이 없어요" />;
    }
    return (
      <div className="flex flex-col gap-2">
        <List className="xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>li:nth-last-child(2):nth-child(odd)]:border-b-0">
          {sessions.map((s) => (
            <ListRow key={s.id} href={checkHref(s.id)} linkAs={Link} title={historyTitle(s)} meta={historyMeta(s)} />
          ))}
        </List>
        <LoadMore hasNextPage={Boolean(history.hasNextPage)} isFetching={history.isFetchingNextPage} onLoad={() => void history.fetchNextPage()} />
      </div>
    );
  }
}
