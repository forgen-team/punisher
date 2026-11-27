import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { EmptyState, ErrorState, ListSkeleton, ReadingLayout, useMediaQuery } from "@forgen-team/ui";
import {
  lectureHref,
  startLectureHref,
  useCheckHistory,
  useConceptDetail,
  useLecture,
  useLectureExercises,
  useLectureLinks,
  useMastery,
  usePrompts,
  useToday,
} from "@/src/entities/punisher";
import { usePageActions, useScreenTitle, type PageAction } from "@/src/shared/lib/screen-chrome";
import { READING_VARS, useUiPrefs } from "@/src/shared/lib/ui-prefs";
import { ExerciseProvider } from "@/src/features/punisher/lecture-exercise";
import { ReadingSignalsProvider } from "@/src/features/punisher/reading-signals";
import { CheckStartButton, CheckStartError, useLaunchCheck } from "@/src/features/punisher/check-start";
import { useLectureProgress } from "@/src/features/punisher/lecture-progress";
import { PretestPanel, usePretest } from "@/src/features/punisher/pretest";
import { ReadingAssist, ReadingThreadsProvider, THREAD_PANEL_QUERY, useReadingThreads } from "@/src/features/punisher/reading-assist";
import { LevelSuggestionCard } from "@/src/features/punisher/level-suggestion";
import { FeedbackBar, LectureWrapup, useLectureActions } from "@/src/features/punisher/lecture-actions";
import type { DocumentLink, Lecture, LectureContent, SectionFix } from "@/src/shared/api/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { DeeperLecturesSection } from "./ui/DeeperLectures";
import { DeepLectureStatus } from "./ui/DeepLectureStatus";
import { FlaggedBanner } from "./ui/FlaggedBanner";
import { latestPublished } from "./lib/newerVersion";
import { LectureBody } from "./ui/LectureBody";
import { LectureMeta } from "./ui/LectureMeta";
import { RegenerateSheet } from "./ui/RegenerateSheet";
import { FixesSheet } from "./ui/FixesSheet";
import { NotesAndQuestions, PanelSheets, ReadingAside, ReadingBar, useAssistCount } from "./ui/LectureSidePanel";
import { SectionToc } from "./ui/SectionToc";
import { sourcesNotInSections } from "./lib/lectureSources";
import { useActiveSection } from "./lib/useActiveSection";
import { useLecturePanel, type LecturePanel } from "./lib/useLecturePanel";
import { earliestReview, lastFinishedCheck, pickNextLecture } from "./lib/wrapupState";
import { SourcesList } from "./ui/blocks";
import { messages } from "./i18n";

const NO_FIXES: SectionFix[] = [];

/**
 * 강의 화면(T-reading, 재설계 §4-2). 화면 제목·←·페이지 동작은 틀(Page)이 두 환경 모두 그린다 —
 * 여기서는 제목(useScreenTitle)과 ⋯ 동작(usePageActions)만 넘긴다.
 * 구성: (보조 열 없을 때) 고정 바 [목차][질문과 메모 n] → 메타 한 줄 → 설명 수준 제안 → 검수 경고·예측 질문 →
 * 본문 → 출처·더 깊게 본 강의 → 끝 마무리 줄 → 피드백 줄. 넓은 화면은 오른쪽 보조 열 탭 [목차][질문과 메모 n].
 */
export function LectureViewer({ lectureId }: { lectureId: string }) {
  const t = useT(messages);
  const lectureQuery = useLecture(lectureId);
  const lecture = lectureQuery.data;
  const content = lecture?.content;
  const { getAnchor, saveNow } = useLectureProgress(lectureId, lecture?.progress, Boolean(content));
  useScreenTitle(content?.title ?? lecture?.title);

  if (lectureQuery.isLoading) {
    return (
      <div aria-busy="true" aria-label={t.loading} className="w-full max-w-[var(--reading-measure,42rem)] pt-4">
        <ListSkeleton rows={6} />
      </div>
    );
  }
  if (lectureQuery.error) {
    return <ErrorState title={t.loadFailed} onRetry={() => void lectureQuery.refetch()} />;
  }
  if (!lecture || !content || content.sections.length === 0) {
    return <EmptyState title={t.emptyTitle} description={t.emptyDescription} />;
  }
  return (
    <ReadingThreadsProvider lectureId={lectureId} conceptId={lecture.conceptId} getAnchor={getAnchor}>
      <LectureReading lecture={lecture} content={content} getAnchor={getAnchor} saveNow={saveNow} />
    </ReadingThreadsProvider>
  );
}

interface LectureReadingProps {
  lecture: Lecture;
  content: LectureContent;
  getAnchor: () => string;
  saveNow: () => void;
}

function LectureReading({ lecture, content, getAnchor, saveNow }: LectureReadingProps) {
  const t = useT(messages);
  const locale = useLocale();
  const lectureId = lecture.id;
  const conceptId = lecture.conceptId;
  const concept = useConceptDetail(conceptId);
  const mastery = useMastery(conceptId);
  const check = useLaunchCheck(lectureId);
  const pretest = usePretest({ lectureId });
  const actions = useLectureActions(lectureId, conceptId, Boolean(lecture.progress?.completedAt), concept.data?.userState);
  const links = useLectureLinks(lectureId);
  const prompts = usePrompts();
  const today = useToday();
  const history = useCheckHistory();
  const fixes = useLectureExercises(lectureId).data?.fixes ?? NO_FIXES;
  const [regenOpen, setRegenOpen] = useState(false);
  const [fixesOpen, setFixesOpen] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);
  const ui = useUiPrefs();
  const readingStyle = { ...READING_VARS[ui.fontSize], ...READING_VARS[ui.lineWidth] } as CSSProperties;

  // 보조 열은 ReadingLayout 이 두 열이 되는 폭(1280)부터 — 스레드도 이 폭부터 패널에서 펼친다.
  const desktop = useMediaQuery(THREAD_PANEL_QUERY);
  const { panel, open: openPanel, close: closePanel } = useLecturePanel();
  const [asideTab, setAsideTab] = useState<LecturePanel>(panel ?? "toc");
  useEffect(() => {
    if (desktop && panel) { setAsideTab(panel); }
  }, [desktop, panel]);
  // 본문에서 새 스레드를 열면(쉽게·깊게·질문·대화 n) 보조 열을 '질문과 메모' 탭으로.
  const openSeq = useReadingThreads()?.openSeq ?? 0;
  useEffect(() => {
    if (openSeq > 0) { setAsideTab("notes"); }
  }, [openSeq]);
  const assistCount = useAssistCount(lectureId);

  // 본문 생성에 쓰는 블록(개요·강사 원칙·형식·감수)을 바꿨을 때만 '다시 만들기'를 보인다.
  const lecturePromptEdited = (prompts.data?.blocks ?? []).some(
    (b) => b.overridden && (b.stage === "outline" || b.stage === "lecture" || b.stage === "review"),
  );

  // 페이지 동작 ⋯(재설계 §3): 이미 앎 · 고칠 곳 n · 새 프롬프트로 다시 만들기 · 지도에서 보기 · 강의 프롬프트.
  const pageActions: PageAction[] = [
    ...(actions.knownDone ? [] : [{ id: "known", label: t.menuKnown, onSelect: actions.markKnown }]),
    ...(fixes.length > 0 ? [{ id: "fixes", label: t.menuFixes(fixes.length), onSelect: () => setFixesOpen(true) }] : []),
    ...(lecturePromptEdited ? [{ id: "regenerate", label: t.menuRegenerate, onSelect: () => setRegenOpen(true) }] : []),
    { id: "map", label: t.menuMap, icon: "map", href: `/map?node=${encodeURIComponent(conceptId)}` },
    { id: "prompts", label: t.menuPrompts, icon: "scroll-text", href: "/settings/prompts" },
  ];
  usePageActions(pageActions);

  const onBeforeDeeper = useCallback(() => {
    saveNow();
    return { lectureId, anchor: getAnchor() };
  }, [saveNow, lectureId, getAnchor]);

  // 섹션별 출처 블록에 이미 나온 링크는 강의 끝에서 다시 나열하지 않는다(남는 것이 없으면 목록을 숨김).
  const sources = useMemo<DocumentLink[]>(
    () => sourcesNotInSections(Array.isArray(lecture.sources) ? lecture.sources : [], content),
    [lecture.sources, content],
  );
  const anchorKey = useMemo(() => content.sections.map((s) => s.anchor).filter(Boolean).join("|"), [content]);
  const activeAnchor = useActiveSection(anchorKey);

  // 표시 시간은 서버가 분량으로 계산한 값(v2.2 §5). 예전 응답(0)이면 섹션 합.
  const totalMinutes = lecture.estimatedMinutes || content.estimatedMinutes;
  const parentLecture = links.data?.parents[0];
  const deeperLectures = (links.data?.children ?? []).filter((l) => l.linkKind !== "regenerated");
  const regenerated = (links.data?.children ?? []).filter((l) => l.linkKind === "regenerated");
  const newerVersion = latestPublished(regenerated);

  const objectiveIds = content.objectiveIds.length > 0 ? content.objectiveIds : lecture.objectiveIds;
  const lastCheck = lastFinishedCheck(history.data?.pages[0]?.sessions, lectureId);
  const reviewAt = lastCheck ? earliestReview(mastery.data, objectiveIds) : null;
  const nextTarget = pickNextLecture(today.data?.items, lectureId, conceptId);
  const next = nextTarget
    ? { title: nextTarget.title, href: nextTarget.lectureId ? lectureHref(nextTarget.lectureId) : startLectureHref(nextTarget.conceptId) }
    : null;

  const toc = <SectionToc sections={content.sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} hideHeading />;
  const notes = <NotesAndQuestions lectureId={lectureId} conceptId={conceptId} getAnchor={getAnchor} />;

  return (
    <div style={readingStyle} className="flex w-full flex-col">
      {!desktop && <ReadingBar count={assistCount} onOpen={openPanel} />}
      <ReadingLayout
        meta={<LectureMeta area={concept.data?.area} minutes={totalMinutes} parent={parentLecture} />}
        aside={desktop ? <ReadingAside tab={asideTab} onTabChange={setAsideTab} count={assistCount} toc={toc} notes={notes} /> : undefined}
        footer={
          <div className="flex flex-col gap-6">
            <LectureWrapup
              actions={actions}
              checkAction={(primary) => <CheckStartButton check={check} primary={primary} />}
              checkStatus={<CheckStartError check={check} />}
              objectiveCount={objectiveIds.length}
              checked={lastCheck ? {
                correct: lastCheck.unaidedCorrect,
                total: lastCheck.itemCount,
                nextReview: reviewAt ? formatDate(reviewAt, locale, { month: "long", day: "numeric" }) : undefined,
              } : null}
              next={next}
            />
            <FeedbackBar lectureId={lectureId} getAnchor={getAnchor} />
          </div>
        }
      >
        <LevelSuggestionCard />
        <DeepLectureStatus lectureId={lectureId} />
        {lecture.reviewState === "flagged" && <FlaggedBanner lectureId={lectureId} reason={lecture.flagReason} reasonCodes={lecture.flagReasonCodes} newerId={newerVersion?.id} />}
        <PretestPanel detail={pretest.data} />
        <div ref={bodyRef}>
          <ReadingSignalsProvider lectureId={lectureId} containerRef={bodyRef}>
            <ExerciseProvider lectureId={lectureId}>
              <LectureBody content={content} onBeforeDeeper={onBeforeDeeper} />
            </ExerciseProvider>
          </ReadingSignalsProvider>
        </div>
        <ReadingAssist containerRef={bodyRef} />
        <SourcesList sources={sources} title={t.sourcesTitle} />
        <DeeperLecturesSection kind="deeper" lectures={deeperLectures} />
        <DeeperLecturesSection
          kind="regenerated"
          lectures={regenerated}
          current={{ id: lectureId, title: content.title || lecture.title, version: lecture.version, createdAt: lecture.createdAt, promptVersion: lecture.promptVersion ?? "" }}
        />
      </ReadingLayout>
      <PanelSheets
        panel={desktop ? null : panel}
        onClose={closePanel}
        sections={content.sections}
        totalMinutes={totalMinutes}
        lectureId={lectureId}
        conceptId={conceptId}
        getAnchor={getAnchor}
      />
      <RegenerateSheet lectureId={lectureId} open={regenOpen} onClose={() => setRegenOpen(false)} />
      <FixesSheet lectureId={lectureId} fixes={fixes} sections={content.sections} open={fixesOpen} onClose={() => setFixesOpen(false)} />
    </div>
  );
}
