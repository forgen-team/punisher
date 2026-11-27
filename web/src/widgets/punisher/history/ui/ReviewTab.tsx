import { EmptyState, ErrorState, List, ListRow, ListSkeleton, Section } from "@forgen-team/ui";
import { describeCheckError, useCards, useCheckHistory } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { useLocale, useT } from "@/src/shared/i18n";
import { historyMeta, historyTitle, matchSessions } from "../lib/checkHistory";
import { messages } from "../i18n";
import { LeechModule } from "./LeechModule";
import { LoadMore } from "./LoadMore";
import { NoMatch } from "./NoMatch";
import { ResolvedModule } from "./ResolvedModule";
import { RetentionModule } from "./RetentionModule";
import { ActivityModules, ManageCardsLink, ProgressModule } from "./StudyModules";
import { ToToday } from "./ToToday";
import { useHistoryNav } from "./useHistoryNav";

/**
 * 복습·통계 탭(T-dash 규칙 — 모듈 머리 + 본문): 위 = 오늘 진행 · 지난 12주 · 앞으로 14일 복습 · 기억 유지,
 * 아래 = 자꾸 막히는 개념 · 지난 복습 · 고친 오개념. 검색어가 있으면 복습 세션만 거른 목록이다.
 */
export function ReviewTab() {
  const { q } = useHistoryNav();
  if (q) { return <SessionResults q={q} />; }
  return (
    <div className="flex flex-col gap-4">
      <ProgressModule />
      <ActivityModules />
      <RetentionModule />
      <LeechModule />
      <PastReviews />
      <ResolvedModule />
    </div>
  );
}

/** 지난 복습 세션(최신순) — 누르면 그 세션 화면(check/[id]). 머리에 [카드 n장 관리]. */
function PastReviews() {
  const locale = useLocale();
  const t = useT(messages);
  const history = useCheckHistory();
  const cards = useCards(false);
  const sessions = history.data?.pages.flatMap((p) => p.sessions) ?? [];
  return (
    <Section variant="card" title={t.pastReviews} action={<ManageCardsLink count={cards.data?.length} />}>
      {renderBody()}
    </Section>
  );

  function renderBody() {
    if (history.isLoading) { return <ListSkeleton rows={3} />; }
    if (history.error) { return <ErrorState title={describeCheckError(history.error)} onRetry={() => void history.refetch()} />; }
    if (sessions.length === 0) { return <EmptyState title={t.emptyReview} action={<ToToday />} />; }
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

function SessionResults({ q }: { q: string }) {
  const locale = useLocale();
  const history = useCheckHistory();
  if (history.isLoading) { return <ListSkeleton rows={4} />; }
  if (history.error) { return <ErrorState title={describeCheckError(history.error)} onRetry={() => void history.refetch()} />; }
  const hits = matchSessions(history.data?.pages.flatMap((p) => p.sessions) ?? [], q, locale);
  if (hits.length === 0) { return <NoMatch />; }
  return (
    <div className="flex flex-col gap-2">
      <List>
        {hits.map((s) => (
          <ListRow key={s.id} href={checkHref(s.id)} title={historyTitle(s, locale)} meta={historyMeta(s, locale)} />
        ))}
      </List>
      <LoadMore hasNextPage={Boolean(history.hasNextPage)} isFetching={history.isFetchingNextPage} onLoad={() => void history.fetchNextPage()} />
    </div>
  );
}
