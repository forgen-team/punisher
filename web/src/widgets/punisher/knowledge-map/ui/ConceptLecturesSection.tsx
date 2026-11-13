
import { Link } from "@/src/shared/lib/router";
import { ErrorState, List, ListRow, ListSkeleton, Section } from "@/src/shared/ui/kit";
import { READ_THROUGH_LABEL, USER_STATE_LABELS, isReadThrough, useConceptLectures } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type ConceptLecture } from "@/src/shared/api/punisher";

function actionLabel(lecture: ConceptLecture): string {
  if (lecture.progressState === "in_progress") { return "이어 보기"; }
  if (lecture.progressState === "completed") { return "다시 보기"; }
  return "열기";
}

/** 진행 상태 — 끝까지 읽고 완료를 안 누른 강의(진행 중 100%)는 '다 읽음'. */
function stateOf(l: ConceptLecture): string {
  if (l.progressState !== "in_progress") { return USER_STATE_LABELS[l.progressState]; }
  if (isReadThrough(l.percent, false)) { return READ_THROUGH_LABEL; }
  return `${USER_STATE_LABELS[l.progressState]} ${l.percent}%`;
}

function metaOf(l: ConceptLecture): string {
  return `${actionLabel(l)} · 깊이 ${l.depth} · ${l.kind === "instant" ? "즉석" : "라이브러리"} · ${stateOf(l)}`;
}

function LectureRows({ conceptId }: { conceptId: string }) {
  const lectures = useConceptLectures(conceptId);
  if (lectures.isLoading) { return <ListSkeleton rows={1} />; }
  if (lectures.error) {
    return <ErrorState compact message={describePunisherError(lectures.error)} onRetry={() => lectures.refetch()} />;
  }
  const items = lectures.data ?? [];
  if (items.length === 0) {
    return <p className="text-xs text-fg-muted">아직 강의가 없어요. 강의 시작으로 만들 수 있어요.</p>;
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
