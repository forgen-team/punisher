
import { Link } from "@/src/shared/lib/router";
import { useCallback, useMemo, useRef, useState } from "react";
import { useAtom } from "jotai";
import { ArrowLeft } from "lucide-react";
import { Button, EmptyState, ErrorState, ListSkeleton, type MenuItem } from "@/src/shared/ui/kit";
import { lectureAnchorHref, lectureBookmarkAtom, useConceptDetail, useLecture, useLectureExercises, useLectureLinks, useMastery, usePrompts } from "@/src/entities/punisher";
import { shellHasActions } from "@/src/shared/config/runtime";
import { usePageActions, useScreenTitle, type PageAction } from "@/src/shared/lib/screen-chrome";
import { ExerciseProvider } from "@/src/features/punisher/lecture-exercise";
import { ReadingSignalsProvider } from "@/src/features/punisher/reading-signals";
import { CheckStartButton, CheckStartError, KnownVerifyChoice, useKnownChoice, useLaunchCheck } from "@/src/features/punisher/check-start";
import { useLectureProgress } from "@/src/features/punisher/lecture-progress";
import { PretestPanel, usePretest } from "@/src/features/punisher/pretest";
import { ReadingAssist, ReadingThreadsProvider, ThreadCountBadge } from "@/src/features/punisher/reading-assist";
import { LevelSuggestionCard } from "@/src/features/punisher/level-suggestion";
import { LectureActionStatus, LectureWrapup, useLectureActions } from "@/src/features/punisher/lecture-actions";
import { describePunisherError, type DocumentLink, type SectionFix } from "@/src/shared/api/punisher";
import { DeeperLecturesSection, ParentLectureLink } from "./ui/DeeperLectures";
import { DeepLectureStatus } from "./ui/DeepLectureStatus";
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
import { messages } from "./i18n";
import { useT } from "@/src/shared/i18n";

/**
 * 강의 뷰어(모바일 우선): 고정 상단 바(목차·질문/메모 시트·⋯) → 제목·메타 → 접힌 목표/범위 → 본문.
 * 넓은 화면 배치는 LectureLayout(lg 접은 목차 · xl 보조 열 · 2xl 펼친 목차). 진도는 스크롤 앵커로 저장·복원된다.
 * ⋯ 메뉴: 이 강의의 상태를 바꾸는 본문 동작(완료·확인·이미 앎·피드백·더 깊은 강의·고칠 점·다시 만들기).
 * 이동·설정(지도에서 보기·강의 프롬프트)은 페이지 동작 — Witems 셸이 그리면(ctx.setActions) 셸 ⋯ 로 넘기고,
 * 단독판·구버전 셸이면 같은 ⋯ 메뉴 아래 구분선 뒤에 링크로 둔다(UI-PAGE-LAYOUT §1, 점검 21).
 */
const NO_FIXES: SectionFix[] = [];

export function LectureViewer({ lectureId }: { lectureId: string }) {
  const t = useT(messages);
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
  const actions = useLectureActions(lectureId, lecture?.conceptId ?? "", Boolean(lecture?.progress?.completedAt), knownChoice, concept.data?.userState);
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
  // 화면 제목은 강의 제목(셸 상단 바·문서 제목 — 점검 22).
  useScreenTitle(content?.title ?? lecture?.title);
  const shellActions = shellHasActions();
  const conceptId = lecture?.conceptId ?? "";
  const pageActions = useMemo<PageAction[]>(
    () => (conceptId
      ? [
          { id: "map", label: t.menuMap, icon: "map", href: `/map?concept=${encodeURIComponent(conceptId)}` },
          { id: "prompts", label: t.menuPrompts, icon: "scroll-text", href: "/settings/prompts" },
        ]
      : []),
    [conceptId, t.menuMap, t.menuPrompts],
  );
  usePageActions(shellActions && pageActions.length > 0 ? pageActions : null);

  if (lectureQuery.isLoading) {
    return (
      <div aria-busy="true" aria-label={t.loading} className="mx-auto w-full max-w-2xl pt-4">
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
          {t.backToLectures}
        </Link>
      </div>
    );
  }
  if (!lecture || !content || content.sections.length === 0) {
    return <EmptyState title={t.emptyTitle} description={t.emptyDescription} />;
  }

  // 표시 시간은 서버가 분량으로 계산한 값(v2.2 §5). 예전 응답(0)이면 섹션 합.
  const totalMinutes = lecture.estimatedMinutes || content.estimatedMinutes;
  const parentLecture = links.data?.parents[0];
  const deeperLectures = (links.data?.children ?? []).filter((l) => l.linkKind !== "regenerated");
  const regenerated = (links.data?.children ?? []).filter((l) => l.linkKind === "regenerated");
  const newerVersion = latestPublished(regenerated);
  // 서버에 기록된 원 강의가 있으면 그것을, 없으면(이전 데이터) 세션 북마크를 쓴다.
  const showReturn = !parentLecture && bookmark && bookmark.lectureId !== lectureId;
  const returnHref = bookmark ? lectureAnchorHref(bookmark.lectureId, bookmark.anchor) : "";

  const menuItems: MenuItem[] = [
    { key: "complete", label: actions.done ? t.menuCompleted : t.menuComplete, disabled: actions.done || actions.complete.isPending, onSelect: actions.markComplete },
    { key: "check", label: t.menuCheck, disabled: check.isPending, onSelect: check.launch },
    { key: "known", label: actions.knownDone ? t.menuKnownDone : t.menuKnown, disabled: actions.knownDone || actions.known.isPending, onSelect: actions.markKnown },
    {
      key: "feedback",
      label: t.menuFeedback,
      onSelect: () => {
        wrapupRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
        wrapupRef.current?.focus({ preventScroll: true });
      },
    },
    ...(deeperLectures.length > 0
      ? [
          {
            key: "deeper",
            label: t.menuDeeper(deeperLectures.length),
            onSelect: () => {
              deeperRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
              deeperRef.current?.focus({ preventScroll: true });
            },
          },
        ]
      : []),
    ...(fixes.length > 0
      ? [{ key: "fixes", label: t.menuFixes(fixes.length), onSelect: () => setFixesOpen(true) }]
      : []),
    ...(lecturePromptEdited
      ? [{ key: "regenerate", label: t.menuRegenerate, onSelect: () => setRegenOpen(true) }]
      : []),
    // 셸이 페이지 동작을 그리지 않으면 이동·설정 링크를 구분선 뒤에 둔다.
    ...(shellActions
      ? []
      : [
          ...pageActions.map((a, i) => ({ key: a.id, label: a.label, href: a.href, separatorBefore: i === 0 })),
          { key: "example-language", label: t.menuExampleLanguage, href: "/settings" },
        ]),
  ];

  return (
    <ReadingThreadsProvider lectureId={lectureId} conceptId={lecture.conceptId} getAnchor={getAnchor}>
    <div className="flex w-full flex-col">
      <LectureTopBar
        title={concept.data ? `${concept.data.name} · ${content.title}` : content.title}
        onOpenToc={() => setTocOpen(true)}
        onOpenAssist={() => setAssistOpen(true)}
        assistBadge={<ThreadCountBadge />}
        menuItems={menuItems}
        menuText={actions.done ? { full: t.menuStatusDone, short: t.menuStatusDone } : { full: t.menuStatus, short: t.menuStatusShort }}
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
        <LevelSuggestionCard />
        <DeepLectureStatus lectureId={lectureId} />
        {parentLecture && <ParentLectureLink parent={parentLecture} />}
        {showReturn && (
          <div className="flex items-center gap-2 border-b border-line py-1 text-sm">
            <Link href={returnHref} className="inline-flex min-h-touch min-w-0 flex-1 items-center gap-1 text-accent-text">
              <ArrowLeft className="size-4 shrink-0" aria-hidden="true" />
              <span className="truncate">{t.returnTo(bookmark.title)}</span>
            </Link>
            <Button variant="ghost" size="sm" onClick={() => setBookmark(null)}>
              {t.clear}
            </Button>
          </div>
        )}
        <LectureHeader lecture={lecture} content={content} concept={concept.data} mastery={mastery.data} minutes={totalMinutes} />
        {lecture.reviewState === "flagged" && <FlaggedBanner lectureId={lectureId} reason={lecture.flagReason} reasonCodes={lecture.flagReasonCodes} newerId={newerVersion?.id} />}
        <PretestPanel detail={pretest.data} />
        <LectureActionStatus actions={actions} />
        <div aria-live="polite" className="empty:hidden">
          <CheckStartError check={check} />
        </div>
        <div ref={bodyRef} className="mt-6">
          <ReadingSignalsProvider lectureId={lectureId} containerRef={bodyRef}>
            <ExerciseProvider lectureId={lectureId}>
              <LectureBody content={content} onBeforeDeeper={onBeforeDeeper} />
            </ExerciseProvider>
          </ReadingSignalsProvider>
        </div>
        <ReadingAssist containerRef={bodyRef} />
        <SourcesList sources={sources} title={t.sourcesTitle} />
        <DeeperLecturesSection ref={deeperRef} kind="deeper" lectures={deeperLectures} />
        <DeeperLecturesSection
          kind="regenerated"
          lectures={regenerated}
          current={{ id: lectureId, title: content.title || lecture.title, version: lecture.version, createdAt: lecture.createdAt, promptVersion: lecture.promptVersion ?? "" }}
        />
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
    </ReadingThreadsProvider>
  );
}
