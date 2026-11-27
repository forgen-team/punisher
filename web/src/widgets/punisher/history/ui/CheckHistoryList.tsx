
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@forgen-team/ui";
import { describeCheckError, useCheckHistory } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { historyMeta, historyTitle } from "../lib/checkHistory";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { LoadMore } from "./LoadMore";
import { ResolvedMisconceptions } from "./ResolvedMisconceptions";
import { RetentionMetrics } from "./RetentionMetrics";

/** 확인·복습 세션(최신순, 한 칸 — 시간순으로 읽힌다) + 아래 기억 유지 지표 · 고친 오개념. 누르면 그 세션 화면(check/[id]). */
export function CheckHistoryList() {
  const locale = useLocale();
  const t = useT(messages);
  const history = useCheckHistory();
  const sessions = history.data?.pages.flatMap((p) => p.sessions) ?? [];
  return (
    <div className="flex flex-col gap-8">
      {renderSessions()}
      <RetentionMetrics />
      <ResolvedMisconceptions />
    </div>
  );

  function renderSessions() {
    if (history.isLoading) { return <ListSkeleton rows={5} />; }
    if (history.error) {
      return <ErrorState title={describeCheckError(history.error)} onRetry={() => history.refetch()} />;
    }
    if (sessions.length === 0) {
      return <EmptyState title={t.noChecks} />;
    }
    return (
      <div className="flex flex-col gap-2">
        <List>
          {sessions.map((s) => (
            <ListRow key={s.id} href={checkHref(s.id)} title={historyTitle(s, locale)} meta={historyMeta(s, locale)} />
          ))}
        </List>
        <LoadMore hasNextPage={Boolean(history.hasNextPage)} isFetching={history.isFetchingNextPage} onLoad={() => void history.fetchNextPage()} />
      </div>
    );
  }
}
