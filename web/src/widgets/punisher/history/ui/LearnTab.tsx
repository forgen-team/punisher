import { Button, Chip, EmptyState, ErrorState, List, ListRow, ListSkeleton, Section } from "@forgen-team/ui";
import { ConceptStateBadge, InquiryStatusIcon, areaLabel, groupByParent, useCompleted, useCompletedAreas, useConcepts, useSearch } from "@/src/entities/punisher";
import { inquiryHref, lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type CompletedItem, type SearchHit } from "@/src/shared/api/punisher";
import { DeeperGroup } from "@/src/shared/ui/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { historyDay } from "../lib/historyDate";
import { hitHref } from "../lib/hits";
import { matchLearnedConcepts } from "../lib/notesFeed";
import { messages } from "../i18n";
import { LoadMore } from "./LoadMore";
import { NoMatch } from "./NoMatch";
import { Snippet } from "./Snippet";
import { ToToday } from "./ToToday";
import { useHistoryNav } from "./useHistoryNav";

function CompletedRow({ item }: { item: CompletedItem }) {
  const locale = useLocale();
  const t = useT(messages);
  const date = historyDay(item.lastActivityAt, locale);
  if (item.kind === "inquiry") {
    return (
      <ListRow
        href={inquiryHref(item.id)}
        leading={<InquiryStatusIcon status="answered" />}
        title={item.title}
        meta={[t.inquiry, item.followupCount ? t.followups(item.followupCount) : "", date].filter(Boolean).join(" · ")}
      />
    );
  }
  return (
    <ListRow
      href={lectureHref(item.id)}
      title={item.title || item.conceptName}
      meta={[item.conceptName && item.conceptName !== item.title ? item.conceptName : "", item.lectureKind === "instant" ? t.instant : t.library, date].filter(Boolean).join(" · ")}
    />
  );
}

/** 영역 칩 줄 — 칩은 하나만 켠다(다시 누르면 끈다). */
function AreaChips({ counts }: { counts: { area: string; count: number }[] }) {
  const t = useT(messages);
  const { area, go } = useHistoryNav();
  const total = counts.reduce((sum, a) => sum + a.count, 0);
  return (
    <div role="group" aria-label={t.areas} className="-mx-4 flex min-h-11 gap-2 overflow-x-auto px-4 py-1 fold:mx-0 fold:flex-wrap fold:px-0">
      <Chip selected={!area} onSelectedChange={() => go({ area: "" })}>
        {t.allAreas} {total}
      </Chip>
      {counts.map((a) => (
        <Chip key={a.area} selected={area === a.area} onSelectedChange={() => go({ area: area === a.area ? "" : a.area })}>
          {areaLabel(a.area)} {a.count}
        </Chip>
      ))}
    </div>
  );
}

/**
 * 학습 탭: 영역 칩 → 들은 강의(완료 날짜순, "더 깊게"는 원 강의 아래) → 배운 개념. 검색어가 있으면 그 범위로 거르고,
 * 강의 본문에서 걸린 곳(검색 API)을 아래에 따로 보인다. 영역·검색어는 서버 쿼리, 주소에 남는다(?area=&q=).
 */
export function LearnTab() {
  const t = useT(messages);
  const { q, area } = useHistoryNav();
  const areas = useCompletedAreas(q);
  const completed = useCompleted({ area: area || undefined, q: q || undefined });
  const concepts = useConcepts();
  const search = useSearch(q);

  const items = completed.data?.pages.flatMap((p) => p.items) ?? [];
  const learned = matchLearnedConcepts(concepts.data ?? [], q, area);
  const bodyHits = (search.data ?? []).filter((hit) => hit.kind === "lecture" && hit.lectureId);

  const failed = areas.error ?? completed.error ?? concepts.error;
  if (failed) { return <ErrorState title={describePunisherError(failed)} onRetry={() => { void areas.refetch(); void completed.refetch(); void concepts.refetch(); }} />; }
  if (areas.isLoading || completed.isLoading || concepts.isLoading) { return <ListSkeleton rows={5} />; }

  const groups = groupByParent(items, (it) => it.id, (it) => it.parentLectureId);
  const empty = items.length === 0 && learned.length === 0 && bodyHits.length === 0;
  const searching = Boolean(q) && search.isLoading;
  return (
    <div className="flex flex-col gap-4">
      <AreaChips counts={areas.data ?? []} />
      {empty && !searching && <LearnEmpty q={q} area={area} />}
      {items.length > 0 && (
        <div className="flex flex-col gap-2">
          <List>
            {groups.map(({ item, children }) => (
              <DeeperGroup key={item.id} parent={<CompletedRow item={item} />}>
                {children.map((c) => <CompletedRow key={c.id} item={c} />)}
              </DeeperGroup>
            ))}
          </List>
          <LoadMore hasNextPage={Boolean(completed.hasNextPage)} isFetching={completed.isFetchingNextPage} onLoad={() => void completed.fetchNextPage()} />
        </div>
      )}
      {learned.length > 0 && (
        <Section title={t.learnedConcepts}>
          <List>
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
        </Section>
      )}
      {q && (
        <BodyHits loading={search.isLoading} error={search.error} onRetry={() => void search.refetch()} hits={bodyHits} q={q} />
      )}
    </div>
  );

}

function LearnEmpty({ q, area }: { q: string; area: string }) {
  const t = useT(messages);
  const { go } = useHistoryNav();
  if (q) { return <NoMatch />; }
  if (area) {
    return <EmptyState title={t.noLecturesInArea} action={<Button variant="secondary" onClick={() => go({ area: "" })}>{t.clearArea}</Button>} />;
  }
  return <EmptyState title={t.emptyLearn} action={<ToToday />} />;
}

function BodyHits({ loading, error, onRetry, hits, q }: { loading: boolean; error: unknown; onRetry: () => void; hits: SearchHit[]; q: string }) {
  const t = useT(messages);
  if (loading) { return <ListSkeleton rows={2} />; }
  if (error) { return <ErrorState title={describePunisherError(error)} onRetry={onRetry} />; }
  if (hits.length === 0) { return null; }
  return (
    <Section title={t.bodyHits}>
      <List>
        {hits.map((hit) => (
          <ListRow
            key={`${hit.refId}:${hit.anchor}`}
            href={hitHref(hit)}
            title={[hit.title, hit.sectionTitle || t.untitledSection].filter(Boolean).join(" · ")}
            metaLines={2}
            meta={<Snippet text={hit.snippet} q={q} />}
          />
        ))}
      </List>
    </Section>
  );
}
