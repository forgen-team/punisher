
import { Link } from "@/src/shared/lib/router";
import { ErrorState, List, ListRow, ListSkeleton, Section } from "@/src/shared/ui/kit";
import { USER_STATE_LABELS, useConceptLectures } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type ConceptLecture } from "@/src/shared/api/punisher";

function actionLabel(lecture: ConceptLecture): string {
  if (lecture.progressState === "in_progress") { return "이어 보기"; }
  if (lecture.progressState === "completed") { return "다시 보기"; }
  return "열기";
}

function metaOf(l: ConceptLecture): string {
  const progress = l.progressState === "in_progress" ? ` ${l.percent}%` : "";
  return `${actionLabel(l)} · 깊이 ${l.depth} · ${l.kind === "instant" ? "즉석" : "라이브러리"} · ${USER_STATE_LABELS[l.progressState]}${progress}`;
}

function LectureRows({ conceptId }: { conceptId: string }) {
  const lectures = useConceptLectures(conceptId);
  if (lectures.isLoading) { return <ListSkeleton rows={1} />; }
  if (lectures.error) {
    return <ErrorState compact message={describePunisherError(lectures.error)} onRetry={() => lectures.refetch()} />;
  }
  const items = lectures.data ?? [];
  if (items.length === 0) {
    return <p className="text-xs text-fg-muted">아직 강의가 없습니다. 강의 시작으로 만들 수 있습니다.</p>;
  }
  return (
    <List>
      {items.map((l) => (
        <ListRow key={l.id} href={lectureHref(l.id)} linkAs={Link} titleLines={1} title={l.title} meta={metaOf(l)} />
      ))}
    </List>
  );
}

/** 이 개념의 강의 목록(라이브러리·내 즉석 강의)과 내 진행 상태. */
export function ConceptLecturesSection({ conceptId }: { conceptId: string }) {
  return (
    <Section title="관련 강의" headingLevel={3}>
      <LectureRows conceptId={conceptId} />
    </Section>
  );
}
