
import { ErrorState, List, ListRow, ListSkeleton, Section } from "@forgen-team/ui";
import { USER_STATE_LABELS, readThroughLabel, isReadThrough, useConceptLectures } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type ConceptLecture } from "@/src/shared/api/punisher";
import { translate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

type T = (typeof messages)["ko"];

function actionLabel(lecture: ConceptLecture, t: T): string {
  if (lecture.progressState === "in_progress") { return t.resume; }
  if (lecture.progressState === "completed") { return t.rewatch; }
  return t.open;
}

/** 진행 상태 — 끝까지 읽고 완료를 안 누른 강의(진행 중 100%)는 '다 읽음'. */
function stateOf(l: ConceptLecture, locale: Locale): string {
  if (l.progressState !== "in_progress") { return USER_STATE_LABELS[l.progressState]; }
  if (isReadThrough(l.percent, false)) { return readThroughLabel(locale); }
  return `${USER_STATE_LABELS[l.progressState]} ${l.percent}%`;
}

function metaOf(l: ConceptLecture, locale: Locale): string {
  const t = translate(messages, locale);
  return `${actionLabel(l, t)} · ${t.depth(l.depth)} · ${l.kind === "instant" ? t.instant : t.library} · ${stateOf(l, locale)}`;
}

function LectureRows({ conceptId }: { conceptId: string }) {
  const lectures = useConceptLectures(conceptId);
  const locale = useLocale();
  const t = translate(messages, locale);
  if (lectures.isLoading) { return <ListSkeleton rows={1} />; }
  if (lectures.error) {
    return <ErrorState title={describePunisherError(lectures.error)} onRetry={() => void lectures.refetch()} />;
  }
  const items = lectures.data ?? [];
  if (items.length === 0) {
    return <p className="text-xs text-fg-muted">{t.lecturesEmpty}</p>;
  }
  return (
    <List>
      {items.map((l) => (
        <ListRow key={l.id} href={lectureHref(l.id)} title={l.title} meta={metaOf(l, locale)} />
      ))}
    </List>
  );
}

/** 이 개념의 강의 목록(라이브러리·내 즉석 강의)과 내 진행 상태. */
export function ConceptLecturesSection({ conceptId }: { conceptId: string }) {
  const t = useT(messages);
  return (
    <Section title={t.lectures} headingLevel={3}>
      <LectureRows conceptId={conceptId} />
    </Section>
  );
}
