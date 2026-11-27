import { Button, Chip, EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@forgen-team/ui";
import { INQUIRY_STATUS_LABELS_BY_LOCALE, inquiryRowTitle, useConcepts, useInquiries, useMyQuestions, useNotes, useSearch } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { historyDay } from "../lib/historyDate";
import { NOTE_KINDS, type NoteKind } from "../lib/historyParams";
import { filterByKind, inquiryRow, memoRow, noteHitHref, noteHits, questionRow, sortNoteRows, type NoteRow } from "../lib/notesFeed";
import { messages } from "../i18n";
import { LoadMore } from "./LoadMore";
import { NoMatch } from "./NoMatch";
import { Snippet } from "./Snippet";
import { ToToday } from "./ToToday";
import { useHistoryNav } from "./useHistoryNav";

const KIND_KEYS = { inquiry: "kindInquiry", lq: "kindLq", memo: "kindMemo" } as const;

function KindChips() {
  const t = useT(messages);
  const { view, go } = useHistoryNav();
  return (
    <div role="group" aria-label={t.kindsLabel} className="-mx-4 flex min-h-11 gap-2 overflow-x-auto px-4 py-1 fold:mx-0 fold:flex-wrap fold:px-0">
      <Chip selected={!view.kind} onSelectedChange={() => go({ kind: "" })}>{t.kindAll}</Chip>
      {NOTE_KINDS.map((kind) => (
        <Chip key={kind} selected={view.kind === kind} onSelectedChange={() => go({ kind: view.kind === kind ? "" : kind })}>
          {t[KIND_KEYS[kind]]}
        </Chip>
      ))}
    </div>
  );
}

/** 질문 해설·강의 중 질문·메모를 한 목록으로(날짜순). 검색어가 있으면 검색 API 결과(발췌 포함)로 바꾼다. */
export function NotesTab() {
  const { q, view } = useHistoryNav();
  return (
    <div className="flex flex-col gap-4">
      <KindChips />
      {q ? <NoteSearchResults q={q} kind={view.kind} /> : <NoteFeed kind={view.kind} />}
    </div>
  );
}

function NoteFeed({ kind }: { kind: NoteKind | "" }) {
  const locale = useLocale();
  const t = useT(messages);
  const { go } = useHistoryNav();
  const inquiries = useInquiries();
  const questions = useMyQuestions();
  const notes = useNotes();
  const concepts = useConcepts();

  const failed = inquiries.error ?? questions.error ?? notes.error;
  if (failed) {
    return <ErrorState title={describePunisherError(failed)} onRetry={() => { void inquiries.refetch(); void questions.refetch(); void notes.refetch(); }} />;
  }
  if (inquiries.isLoading || questions.isLoading || notes.isLoading) { return <ListSkeleton rows={5} />; }

  const questionItems = questions.data?.pages.flatMap((p) => p.items) ?? [];
  // 메모에는 강의 이름이 없다 — 같은 강의에서 한 질문이나 메모가 붙은 개념의 이름을 쓴다.
  const lectureNames = new Map(questionItems.filter((q) => q.lectureId).map((q) => [q.lectureId ?? "", q.lectureTitle || q.conceptName]));
  const conceptNames = new Map((concepts.data ?? []).map((c) => [c.id, c.name]));
  const day = (value: string) => historyDay(value, locale);
  const all = sortNoteRows([
    ...(inquiries.data ?? []).map((i) => inquiryRow(i, inquiryRowTitle(i), INQUIRY_STATUS_LABELS_BY_LOCALE[locale][i.status] ?? i.status, day(i.createdAt))),
    ...questionItems.map((q) => questionRow(q, day(q.createdAt))),
    ...(notes.data ?? []).map((n) => memoRow(n, lectureNames.get(n.lectureId ?? "") ?? conceptNames.get(n.conceptId ?? "") ?? "", day(n.createdAt))),
  ]);
  const rows = filterByKind(all, kind);

  if (rows.length === 0) {
    if (kind && all.length > 0) {
      return <EmptyState title={t.noNotesOfKind} action={<Button variant="secondary" onClick={() => go({ kind: "" })}>{t.clearKind}</Button>} />;
    }
    return <EmptyState title={t.emptyNotes} action={<ToToday />} />;
  }
  const kindLabel = (row: NoteRow) => t[KIND_KEYS[row.kind]];
  return (
    <div className="flex flex-col gap-2">
      <List>
        {rows.map((row) => (
          <ListRow key={row.key} href={row.href} title={row.title} meta={[kindLabel(row), ...row.parts].filter(Boolean).join(" · ")} />
        ))}
      </List>
      {(kind === "" || kind === "lq") && (
        <LoadMore hasNextPage={Boolean(questions.hasNextPage)} isFetching={questions.isFetchingNextPage} onLoad={() => void questions.fetchNextPage()} />
      )}
    </div>
  );
}

function NoteSearchResults({ q, kind }: { q: string; kind: NoteKind | "" }) {
  const t = useT(messages);
  const search = useSearch(q);
  if (search.isLoading) { return <ListSkeleton rows={4} />; }
  if (search.error) { return <ErrorState title={describePunisherError(search.error)} onRetry={() => void search.refetch()} />; }
  const hits = filterByKind(noteHits(search.data ?? []), kind);
  if (hits.length === 0) { return <NoMatch />; }
  return (
    <List>
      {hits.map(({ hit, kind: hitKind }) => (
        <ListRow
          key={`${hit.kind}:${hit.refId}`}
          href={noteHitHref(hit, hitKind)}
          title={hit.title || t[KIND_KEYS[hitKind]]}
          metaLines={2}
          meta={
            <>
              <span className="mr-1 font-medium text-fg-secondary">{t[KIND_KEYS[hitKind]]}</span>
              <Snippet text={hit.snippet} q={q} />
            </>
          }
        />
      ))}
    </List>
  );
}
