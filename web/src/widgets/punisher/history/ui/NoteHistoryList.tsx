
import { Link } from "@/src/shared/lib/router";
import { useMemo } from "react";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { useNotes } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type Note } from "@/src/shared/api/punisher";
import { translate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { historyDay } from "../lib/historyDate";
import { messages } from "../i18n";


function metaOf(n: Note, locale: Locale): string {
  return [historyDay(n.createdAt, locale), n.quote ? translate(messages, locale).quote(n.quote) : ""].filter(Boolean).join(" · ");
}

/** 메모 전체(최신순). 강의 메모는 눌러서 해당 위치로 이동. */
export function NoteHistoryList() {
  const locale = useLocale();
  const t = useT(messages);
  const notes = useNotes();
  const sorted = useMemo(() => [...(notes.data ?? [])].sort((a, b) => b.createdAt.localeCompare(a.createdAt)), [notes.data]);
  if (notes.isLoading) { return <ListSkeleton rows={5} />; }
  if (notes.error) {
    return <ErrorState message={describePunisherError(notes.error)} onRetry={() => notes.refetch()} />;
  }
  if (sorted.length === 0) {
    return <EmptyState title={t.noNotes} description={t.noNotesDescription} />;
  }
  // 넓은 화면(xl)은 두 열 — 짧은 행이 화면 끝까지 늘어나지 않게.
  return (
    <List className="xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>li:nth-last-child(2):nth-child(odd)]:border-b-0">
      {sorted.map((n) =>
        n.lectureId ? (
          <ListRow
            key={n.id}
            href={`${lectureHref(n.lectureId)}${n.anchor ? `#pa-${n.anchor}` : ""}`}
            linkAs={Link}
            title={n.body || n.quote}
            meta={metaOf(n, locale)}
          />
        ) : (
          <ListRow key={n.id} title={n.body || n.quote} meta={metaOf(n, locale)} />
        ),
      )}
    </List>
  );
}
