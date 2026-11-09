
import { Link } from "@/src/shared/lib/router";
import { useRouter } from "@/src/shared/lib/router";
import { useCallback, useMemo, useRef, useState } from "react";
import { useAtom } from "jotai";
import { ArrowLeft } from "lucide-react";
import { Button, EmptyState, ErrorState, ListSkeleton, type MenuItem } from "@/src/shared/ui/kit";
import { lectureBookmarkAtom, useConceptDetail, useLecture } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { useLectureProgress } from "@/src/features/punisher/lecture-progress";
import { SelectionExplain } from "@/src/features/punisher/reading-assist";
import { LectureActionStatus, LectureWrapup, useLectureActions } from "@/src/features/punisher/lecture-actions";
import { describePunisherError, type DocumentLink } from "@/src/shared/api/punisher";
import { LectureBody } from "./ui/LectureBody";
import { LectureHeader } from "./ui/LectureHeader";
import { LectureAssistSheet } from "./ui/LectureSidePanel";
import { LectureTopBar } from "./ui/LectureTopBar";
import { SectionToc, TocSheet } from "./ui/SectionToc";
import { SourcesList } from "./ui/blocks";

/**
 * 강의 뷰어(모바일 우선): 고정 상단 바(목차·질문/메모 시트·⋯) → 제목·메타 → 접힌 목표/범위 → 본문.
 * lg 이상은 왼쪽에 목차를 고정한다. 진도는 스크롤 앵커로 저장·복원된다.
 */
export function LectureViewer({ lectureId }: { lectureId: string }) {
  const router = useRouter();
  const lectureQuery = useLecture(lectureId);
  const lecture = lectureQuery.data;
  const content = lecture?.content;
  const concept = useConceptDetail(lecture?.conceptId);
  const [bookmark, setBookmark] = useAtom(lectureBookmarkAtom);
  const { getAnchor, saveNow } = useLectureProgress(lectureId, lecture?.progress, Boolean(content));
  const actions = useLectureActions(lectureId, lecture?.conceptId ?? "", Boolean(lecture?.progress?.completedAt));
  const [tocOpen, setTocOpen] = useState(false);
  const [assistOpen, setAssistOpen] = useState(false);
  const wrapupRef = useRef<HTMLElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  const onBeforeDeeper = useCallback(() => {
    saveNow();
    setBookmark({ lectureId, anchor: getAnchor(), title: content?.title ?? lecture?.title ?? "" });
  }, [saveNow, setBookmark, lectureId, getAnchor, content?.title, lecture?.title]);

  const sources = useMemo<DocumentLink[]>(() => (Array.isArray(lecture?.sources) ? lecture.sources : []), [lecture?.sources]);

  if (lectureQuery.isLoading) {
    return (
      <div aria-busy="true" aria-label="강의를 불러오는 중" className="mx-auto w-full max-w-2xl pt-4">
        <ListSkeleton rows={6} />
      </div>
    );
  }
  if (lectureQuery.error) {
    return <ErrorState message={describePunisherError(lectureQuery.error)} onRetry={() => lectureQuery.refetch()} />;
  }
  if (!lecture || !content || content.sections.length === 0) {
    return <EmptyState title="강의 본문이 없습니다" description="강의가 아직 생성 중이거나 비어 있습니다." />;
  }

  const totalMinutes = content.estimatedMinutes || lecture.estimatedMinutes;
  const showReturn = bookmark && bookmark.lectureId !== lectureId;
  const returnHref = bookmark ? `${lectureHref(bookmark.lectureId)}#pa-${bookmark.anchor}` : "";

  const menuItems: MenuItem[] = [
    { key: "complete", label: actions.done ? "완료함" : "완료로 표시", disabled: actions.done || actions.complete.isPending, onSelect: actions.markComplete },
    { key: "known", label: actions.knownDone ? "이미 앎으로 표시됨" : "이미 앎", disabled: actions.knownDone || actions.known.isPending, onSelect: actions.markKnown },
    {
      key: "feedback",
      label: "피드백 남기기",
      onSelect: () => {
        wrapupRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
        wrapupRef.current?.focus({ preventScroll: true });
      },
    },
    {
      key: "example-language",
      label: "예시 코드 언어 바꾸기",
      onSelect: () => router.push("/settings"),
    },
    {
      key: "map",
      label: "지도에서 보기",
      onSelect: () => router.push(`/map?concept=${encodeURIComponent(lecture.conceptId)}`),
    },
  ];

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar
        title={concept.data ? `${concept.data.name} · ${content.title}` : content.title}
        onOpenToc={() => setTocOpen(true)}
        onOpenAssist={() => setAssistOpen(true)}
        menuItems={menuItems}
      />
      <div className="flex gap-8">
        <aside className="sticky top-[calc(var(--app-header-offset)+4rem)] hidden max-h-[calc(100dvh-6rem)] w-56 shrink-0 self-start overflow-y-auto pt-4 lg:block">
          <SectionToc sections={content.sections} totalMinutes={totalMinutes} />
        </aside>
        <article className="mx-auto w-full min-w-0 max-w-2xl pb-16">
          {showReturn && (
            <div className="flex items-center gap-2 border-b border-line py-1 text-sm">
              <Link href={returnHref} className="inline-flex min-h-touch min-w-0 flex-1 items-center gap-1 text-accent-text">
                <ArrowLeft className="size-4 shrink-0" aria-hidden="true" />
                <span className="truncate">읽던 강의로 돌아가기: {bookmark.title}</span>
              </Link>
              <Button variant="ghost" size="sm" onClick={() => setBookmark(null)}>
                지우기
              </Button>
            </div>
          )}
          <LectureHeader lecture={lecture} content={content} concept={concept.data} />
          <LectureActionStatus actions={actions} />
          <div ref={bodyRef} className="mt-6">
            <LectureBody content={content} onBeforeDeeper={onBeforeDeeper} />
          </div>
          <SelectionExplain lectureId={lectureId} containerRef={bodyRef} />
          <SourcesList sources={sources} title="이 강의의 출처" />
          <div className="mt-10">
            <LectureWrapup ref={wrapupRef} lectureId={lectureId} actions={actions} getAnchor={getAnchor} />
          </div>
        </article>
      </div>
      <TocSheet open={tocOpen} onClose={() => setTocOpen(false)} sections={content.sections} totalMinutes={totalMinutes} />
      <LectureAssistSheet
        open={assistOpen}
        onClose={() => setAssistOpen(false)}
        lectureId={lectureId}
        conceptId={lecture.conceptId}
        getAnchor={getAnchor}
      />
    </div>
  );
}
