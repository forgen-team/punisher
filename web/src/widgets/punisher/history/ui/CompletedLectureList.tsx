
import { Link } from "@/src/shared/lib/router";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { groupByParent, useMyLectures } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type MyLecture } from "@/src/shared/api/punisher";
import { DeeperGroup } from "@/src/shared/ui/punisher";
import { LoadMore } from "./LoadMore";

const dateFormat = new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium" });

function metaOf(l: MyLecture): string {
  return [
    l.title && l.title !== l.conceptName ? l.title : "",
    `깊이 ${l.depth}`,
    l.kind === "instant" ? "즉석" : "라이브러리",
    l.completedAt ? `${dateFormat.format(new Date(l.completedAt))} 완료` : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

function LectureRow({ lecture }: { lecture: MyLecture }) {
  return <ListRow href={lectureHref(lecture.id)} linkAs={Link} title={lecture.conceptName || lecture.title} meta={metaOf(lecture)} />;
}

/** 완료한 강의("더 깊게" 본 강의는 원 강의 아래로 묶음): 개념 이름(안정적) / 강의 제목 · 깊이 · 종류 · 완료일. */
export function CompletedLectureList() {
  const lectures = useMyLectures("completed");
  const items = lectures.data?.pages.flatMap((p) => p.items) ?? [];
  if (lectures.isLoading) { return <ListSkeleton rows={5} />; }
  if (lectures.error) {
    return <ErrorState message={describePunisherError(lectures.error)} onRetry={() => lectures.refetch()} />;
  }
  if (items.length === 0) {
    return <EmptyState title="아직 완료한 강의가 없습니다" description="강의를 끝까지 읽고 완료하면 여기에 쌓입니다." />;
  }
  return (
    <div className="flex flex-col gap-3">
      <List>
        {groupByParent(items, (l) => l.id, (l) => l.parentLectureId).map(({ item, children }) => (
          <DeeperGroup key={item.id} parent={<LectureRow lecture={item} />}>
            {children.map((c) => (
              <LectureRow key={c.id} lecture={c} />
            ))}
          </DeeperGroup>
        ))}
      </List>
      <LoadMore hasNextPage={Boolean(lectures.hasNextPage)} isFetching={lectures.isFetchingNextPage} onLoad={() => lectures.fetchNextPage()} />
    </div>
  );
}
