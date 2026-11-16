
import { Link } from "@/src/shared/lib/router";
import { useRouter } from "@/src/shared/lib/router";
import { useCallback, useMemo, useRef, useState } from "react";
import { useAtom } from "jotai";
import { ArrowLeft } from "lucide-react";
import { Button, EmptyState, ErrorState, ListSkeleton, type MenuItem } from "@/src/shared/ui/kit";
import { lectureBookmarkAtom, useConceptDetail, useLecture, useLectureExercises, useLectureLinks, useMastery, usePrompts } from "@/src/entities/punisher";
import { ExerciseProvider } from "@/src/features/punisher/lecture-exercise";
import { CheckStartButton, CheckStartError, KnownVerifyChoice, useKnownChoice, useLaunchCheck } from "@/src/features/punisher/check-start";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { useLectureProgress } from "@/src/features/punisher/lecture-progress";
import { PretestPanel, usePretest } from "@/src/features/punisher/pretest";
import { SelectionExplain } from "@/src/features/punisher/reading-assist";
import { LectureActionStatus, LectureWrapup, useLectureActions } from "@/src/features/punisher/lecture-actions";
import { describePunisherError, type DocumentLink, type SectionFix } from "@/src/shared/api/punisher";
import { DeeperLecturesSection, ParentLectureLink } from "./ui/DeeperLectures";
import { FlaggedBanner } from "./ui/FlaggedBanner";
import { latestPublished } from "./lib/newerVersion";
import { LectureBody } from "./ui/LectureBody";
import { RegenerateSheet } from "./ui/RegenerateSheet";
import { FixesSheet } from "./ui/FixesSheet";
import { LectureHeader } from "./ui/LectureHeader";
import { LectureAssistPanel, LectureAssistSheet } from "./ui/LectureSidePanel";
import { LectureLayout } from "./ui/LectureLayout";
import { LectureTopBar } from "./ui/LectureTopBar";
import { TocSheet } from "./ui/SectionToc";
import { extractLectureTerms } from "./lib/lectureTerms";
import { sourcesNotInSections } from "./lib/lectureSources";
import { useActiveSection } from "./lib/useActiveSection";
import { SourcesList } from "./ui/blocks";

/**
 * 강의 뷰어(모바일 우선): 고정 상단 바(목차·질문/메모 시트·⋯) → 제목·메타 → 접힌 목표/범위 → 본문.
 * 넓은 화면 배치는 LectureLayout(lg 접은 목차 · xl 보조 열 · 2xl 펼친 목차). 진도는 스크롤 앵커로 저장·복원된다.
 */
const NO_FIXES: SectionFix[] = [];

export function LectureViewer({ lectureId }: { lectureId: string }) {
  const router = useRouter();
  const lectureQuery = useLecture(lectureId);
  const lecture = lectureQuery.data;
  const content = lecture?.content;
  const concept = useConceptDetail(lecture?.conceptId);
  const mastery = useMastery(lecture?.conceptId);
  const check = useLaunchCheck(lectureId);
  const pretest = usePretest({ lectureId });
  const [bookmark, setBookmark] = useAtom(lectureBookmarkAtom);
  const { getAnchor, saveNow } = useLectureProgress(lectureId, lecture?.progress, Boolean(content));
  const knownChoice = useKnownChoice();
  const actions = useLectureActions(lectureId, lecture?.conceptId ?? "", Boolean(lecture?.progress?.completedAt), knownChoice);
  const [tocOpen, setTocOpen] = useState(false);
  const [assistOpen, setAssistOpen] = useState(false);
  const wrapupRef = useRef<HTMLElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const deeperRef = useRef<HTMLElement>(null);
  const links = useLectureLinks(lectureId);
  const prompts = usePrompts();
  const [regenOpen, setRegenOpen] = useState(false);
  const [fixesOpen, setFixesOpen] = useState(false);
  const fixes = useLectureExercises(lectureId).data?.fixes ?? NO_FIXES;
  // 본문 생성에 쓰는 블록(개요·강사 원칙·형식·감수)을 바꿨을 때만 '다시 만들기'를 보인다.
  const lecturePromptEdited = (prompts.data?.blocks ?? []).some(
    (b) => b.overridden && (b.stage === "outline" || b.stage === "lecture" || b.stage === "review"),
  );

  const onBeforeDeeper = useCallback(() => {
    saveNow();
    const anchor = getAnchor();
    setBookmark({ lectureId, anchor, title: content?.title ?? lecture?.title ?? "" });
    return { lectureId, anchor };
  }, [saveNow, setBookmark, lectureId, getAnchor, content?.title, lecture?.title]);

  // 섹션별 출처 블록에 이미 나온 링크는 강의 끝에서 다시 나열하지 않는다(남는 것이 없으면 목록을 숨김).
  const sources = useMemo<DocumentLink[]>(
    () => sourcesNotInSections(Array.isArray(lecture?.sources) ? lecture.sources : [], content ?? { sections: [] }),
    [lecture?.sources, content],
  );
  const terms = useMemo(() => (content ? extractLectureTerms(content) : []), [content]);
  const anchorKey = useMemo(() => (content?.sections ?? []).map((s) => s.anchor).filter(Boolean).join("|"), [content]);
  const activeAnchor = useActiveSection(anchorKey);

  if (lectureQuery.isLoading) {
    return (
      <div aria-busy="true" aria-label="강의를 불러오는 중" className="mx-auto w-full max-w-2xl pt-4">
        <ListSkeleton rows={6} />
      </div>
    );
  }
  if (lectureQuery.error) {
    // 단독판: 셸 상단 바에 뒤로 버튼이 없어, 강의를 못 불러오면 강의 화면으로 돌아갈 길을 함께 둔다(docs/SYNC.md).
    return (
      <div className="flex flex-col items-center gap-2">
        <ErrorState message={describePunisherError(lectureQuery.error)} onRetry={() => lectureQuery.refetch()} />
        <Link href="/" className="text-sm text-fg-secondary underline-offset-2 hover:text-fg hover:underline">
          강의 화면으로 돌아가기
        </Link>
      </div>
    );
  }
  if (!lecture || !content || content.sections.length === 0) {
    return <EmptyState title="강의 본문이 없습니다" description="강의가 아직 생성 중이거나 비어 있습니다." />;
  }

  // 표시 시간은 서버가 분량으로 계산한 값(v2.2 §5). 예전 응답(0)이면 섹션 합.
  const totalMinutes = lecture.estimatedMinutes || content.estimatedMinutes;
  const parentLecture = links.data?.parents[0];
  const deeperLectures = (links.data?.children ?? []).filter((l) => l.linkKind !== "regenerated");
  const regenerated = (links.data?.children ?? []).filter((l) => l.linkKind === "regenerated");
  const newerVersion = latestPublished(regenerated);
  // 서버에 기록된 원 강의가 있으면 그것을, 없으면(이전 데이터) 세션 북마크를 쓴다.
  const showReturn = !parentLecture && bookmark && bookmark.lectureId !== lectureId;
  const returnHref = bookmark ? `${lectureHref(bookmark.lectureId)}#pa-${bookmark.anchor}` : "";

  const menuItems: MenuItem[] = [
    { key: "complete", label: actions.done ? "완료함" : "완료로 표시", disabled: actions.done || actions.complete.isPending, onSelect: actions.markComplete },
    { key: "check", label: "확인하기", disabled: check.isPending, onSelect: check.launch },
    { key: "known", label: actions.knownDone ? "이미 앎으로 표시됨" : "이미 앎", disabled: actions.knownDone || actions.known.isPending, onSelect: actions.markKnown },
    {
      key: "feedback",
      label: "피드백 남기기",
      onSelect: () => {
        wrapupRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
        wrapupRef.current?.focus({ preventScroll: true });
      },
    },
    ...(deeperLectures.length > 0
      ? [
          {
            key: "deeper",
            label: `더 깊게 본 강의 ${deeperLectures.length}개`,
            onSelect: () => {
              deeperRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
              deeperRef.current?.focus({ preventScroll: true });
            },
          },
        ]
      : []),
    ...(fixes.length > 0
      ? [{ key: "fixes", label: `고칠 곳 ${fixes.length}`, onSelect: () => setFixesOpen(true) }]
      : []),
    ...(lecturePromptEdited
      ? [{ key: "regenerate", label: "이 강의 새 프롬프트로 다시 만들기", onSelect: () => setRegenOpen(true) }]
      : []),
    {
      key: "prompts",
      label: "강의 프롬프트 보기·고치기",
      onSelect: () => router.push("/settings/prompts"),
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
      <LectureLayout
        sections={content.sections}
        totalMinutes={totalMinutes}
        activeAnchor={activeAnchor}
        assist={
          <LectureAssistPanel
            lectureId={lectureId}
            conceptId={lecture.conceptId}
            getAnchor={getAnchor}
            terms={terms}
            onBeforeDeeper={onBeforeDeeper}
            deeper={deeperLectures}
          />
        }
      >
        {parentLecture && <ParentLectureLink parent={parentLecture} />}
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
        <LectureHeader lecture={lecture} content={content} concept={concept.data} mastery={mastery.data} minutes={totalMinutes} />
        {lecture.reviewState === "flagged" && <FlaggedBanner lectureId={lectureId} reason={lecture.flagReason} newerId={newerVersion?.id} />}
        <PretestPanel detail={pretest.data} />
        <LectureActionStatus actions={actions} />
        <div aria-live="polite" className="empty:hidden">
          <CheckStartError check={check} />
        </div>
        <div ref={bodyRef} className="mt-6">
          <ExerciseProvider lectureId={lectureId}>
            <LectureBody content={content} onBeforeDeeper={onBeforeDeeper} />
          </ExerciseProvider>
        </div>
        <SelectionExplain lectureId={lectureId} containerRef={bodyRef} />
        <SourcesList sources={sources} title="이 강의의 출처" />
        <DeeperLecturesSection ref={deeperRef} kind="deeper" lectures={deeperLectures} />
        <DeeperLecturesSection kind="regenerated" lectures={regenerated} />
        <div className="mt-10">
          <LectureWrapup
            ref={wrapupRef}
            lectureId={lectureId}
            actions={actions}
            getAnchor={getAnchor}
            checkAction={<CheckStartButton check={check} primary={actions.done} />}
            checkStatus={<CheckStartError check={check} />}
          />
        </div>
      </LectureLayout>
      <TocSheet open={tocOpen} onClose={() => setTocOpen(false)} sections={content.sections} totalMinutes={totalMinutes} />
      <RegenerateSheet lectureId={lectureId} open={regenOpen} onClose={() => setRegenOpen(false)} />
      <FixesSheet lectureId={lectureId} fixes={fixes} sections={content.sections} open={fixesOpen} onClose={() => setFixesOpen(false)} />
      <KnownVerifyChoice choice={knownChoice} />
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
