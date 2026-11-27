import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
import { Button, ErrorState, ProgressCircle, Skeleton, useConfirm, useMediaQuery } from "@forgen-team/ui";
import { usePageActions, useScreenTitle, type PageAction } from "@/src/shared/lib/screen-chrome";
import { useInquiry, useRegenerateInquiry, useRetryFollowup, useSetInquiryRead } from "@/src/entities/punisher";
import { INQUIRY_MAX_FOLLOWUPS, describePunisherError, type Inquiry, type InquiryFollowup, type LectureSection } from "@/src/shared/api/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { THREAD_PANEL_QUERY } from "@/src/features/punisher/reading-assist";
import { FOLLOWUP_ANNOUNCEMENTS, messages } from "./i18n";
import { LectureAssistPlaceholder, LectureBody, LectureLayout, useActiveSection } from "../lecture-viewer";
import { useLecturePanel } from "../lecture-viewer/lib/useLecturePanel";
import { InquiryConcepts, mapHref } from "./ui/InquiryConcepts";
import { FollowupComposer } from "./ui/FollowupComposer";
import { InquiryHeader } from "./ui/InquiryHeader";
import { InquiryTocBar, InquiryTocSheet } from "./ui/InquiryToc";
import { QuoteAsk } from "./ui/QuoteAsk";
import { FollowupQuestion, inquirySectionNote } from "./ui/sectionNote";

const NO_SECTIONS: LectureSection[] = [];
const NO_FOLLOWUPS: InquiryFollowup[] = [];
const ANSWERED_ON: Intl.DateTimeFormatOptions = { month: "long", day: "numeric" };

/**
 * 해설 상태·메타 줄 — 항상 같은 자리·같은 높이(쓰는 중 ↔ 완료가 바뀌어도 본문이 밀리지 않음)이고, aria-busy 밖의
 * 라이브 영역이라 완료도 읽어 준다. 끝난 해설은 "질문 해설 · 날짜 · 약 n분".
 */
function StatusRow({ inquiry, hasContent, unstable, minutes }: { inquiry: Inquiry | undefined; hasContent: boolean; unstable: boolean; minutes: number }) {
  const locale = useLocale();
  const t = useT(messages);
  let content: ReactNode = <Skeleton className="w-40" />;
  if (inquiry?.status === "pending") {
    content = (
      <>
        <ProgressCircle size="24" tone="inherit" aria-hidden />
        {unstable && t.unstable}
        {!unstable && (hasContent ? t.rewriting : t.writing)}
      </>
    );
  } else if (inquiry?.status === "failed") {
    content = (
      <>
        <AlertTriangle className="size-4 shrink-0 text-warning" aria-hidden="true" />
        {t.failed}
      </>
    );
  } else if (inquiry) {
    content = t.meta(inquiry.answeredAt ? formatDate(new Date(inquiry.answeredAt), locale, ANSWERED_ON) : "", minutes);
  }
  return (
    <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
      {content}
    </p>
  );
}

/**
 * '이 질문에 답하기' 해설 화면(T-reading) — 제목·←·페이지 동작(다시 만들기·지도에서 보기)은 틀이 그린다.
 * 접은/펼친 목차 · 가독 폭 본문 · xl 보조 열, 끝 마무리 줄 [읽음으로 표시][이어서 묻기].
 * 보조 열이 없는 폭(강의 화면과 같은 1280 미만)에서는 본문 위 고정 바 [목차] → 시트(주소 ?panel=toc, 뒤로 = 닫기).
 * 보조 열에는 해설에 나온 개념(지도에 있던 것 · 새로 제안한 것). 쓰는 중이면 3초마다 다시 확인하고,
 * 끝난 해설은 저장된 그대로 보인다('다시 만들기'에서만 새로 쓴다). 불러오는 중에도 같은 뼈대를 그려 밀림이 없다.
 */
export function InquiryView({ inquiryId }: { inquiryId: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const inquiry = useInquiry(inquiryId);
  const regenerate = useRegenerateInquiry();
  const setRead = useSetInquiryRead();
  const retryFollowup = useRetryFollowup(inquiryId);
  const confirm = useConfirm();
  const [quote, setQuote] = useState("");
  const bodyRef = useRef<HTMLDivElement>(null);
  const composerRef = useRef<HTMLTextAreaElement>(null);
  const q = inquiry.data;
  const baseSections = q?.content?.sections ?? NO_SECTIONS;
  const followups = q?.followups ?? NO_FOLLOWUPS;
  // 스레드는 한 문서: 해설 섹션 뒤에 답을 받은 후속 문답을 섹션으로 잇는다(목차·진행 표시도 같이).
  const sections = useMemo(
    () => [...baseSections, ...followups.flatMap((f) => (f.status === "answered" && f.section ? [f.section] : []))],
    [baseSections, followups],
  );
  const followupByAnchor = useMemo(
    () => new Map(followups.flatMap((f) => (f.section?.anchor ? ([[f.section.anchor, f]] as const) : []))),
    [followups],
  );
  const sectionNote = useCallback((section: LectureSection) => inquirySectionNote(section, followupByAnchor, locale), [followupByAnchor, locale]);
  const unanswered = followups.filter((f) => f.status !== "answered");
  const followupBusy = unanswered.some((f) => f.status === "pending");
  const lastFollowup = followups[followups.length - 1];
  const followupAnnouncement = lastFollowup ? FOLLOWUP_ANNOUNCEMENTS[locale][lastFollowup.status] : "";
  const pickQuote = useCallback((text: string) => {
    setQuote(text);
    composerRef.current?.focus();
    composerRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, []);
  const anchorKey = useMemo(() => sections.map((s) => s.anchor).filter(Boolean).join("|"), [sections]);
  const activeAnchor = useActiveSection(anchorKey);
  useScreenTitle(q?.content?.title || q?.question);
  const wide = useMediaQuery(THREAD_PANEL_QUERY);
  const { panel, open: openPanel, close: closePanel } = useLecturePanel();

  const hasContent = baseSections.length > 0;
  const generating = q?.status === "pending";
  const read = Boolean(q?.readAt);
  const totalMinutes = sections.reduce((sum, s) => sum + (s.minutes || 0), 0);
  // 해설이 있으면 앱 안 확인 창으로 무엇이 바뀌고 남는지 보인 뒤 다시 만든다(기본 포커스 취소).
  const askRegenerate = async () => {
    if (!q) { return; }
    const ok = !hasContent || (await confirm({
      title: t.confirmTitle,
      description: (
        <>
          {t.confirmDescription}
          <ul className="mt-2 list-disc pl-4">
            {[t.confirmReplace, t.confirmFollowups, t.confirmFailure, t.confirmUsage].map((d) => <li key={d}>{d}</li>)}
          </ul>
        </>
      ),
      confirmLabel: t.regenerate,
      cancelLabel: t.cancel,
      tone: "neutral",
    }));
    if (ok) { regenerate.mutate(q.id); }
  };
  const toggleRead = () => q && setRead.mutate({ id: q.id, read: !read });

  const relatedId = q?.concepts?.find((c) => c.kind === "related")?.id ?? q?.conceptIds[0];
  const pageActions: PageAction[] = [];
  if (q && !generating && !followupBusy && !regenerate.isPending) {
    pageActions.push({ id: "regenerate", label: t.regenerate, onSelect: () => void askRegenerate() });
  }
  if (relatedId) { pageActions.push({ id: "map", label: t.viewOnMap, href: mapHref(relatedId) }); }
  usePageActions(pageActions.length > 0 ? pageActions : null);

  // 이어서 묻기: 본문 끝의 입력란으로 가서 쓰게 한다.
  const askMore = () => {
    composerRef.current?.focus();
    composerRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  };

  // 처음 불러오기에 실패했을 때만 오류 화면. 폴링 중 일시 오류는 본문을 지우지 않고 아래에 알린다.
  if (!q && inquiry.error) {
    return <ErrorState title={describePunisherError(inquiry.error)} onRetry={() => void inquiry.refetch()} />;
  }

  let assist: ReactNode = <LectureAssistPlaceholder>{t.assistPending}</LectureAssistPlaceholder>;
  if (q?.concepts && q.concepts.length > 0) { assist = <InquiryConcepts concepts={q.concepts} />; }
  else if (q && !generating) { assist = <LectureAssistPlaceholder>{t.assistEmpty}</LectureAssistPlaceholder>; }

  return (
    <div className="flex w-full flex-col">
      {!wide && <InquiryTocBar disabled={sections.length === 0} onOpen={() => openPanel("toc")} />}
      <LectureLayout sections={sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} assist={assist}>
        <StatusRow inquiry={q} hasContent={hasContent} unstable={Boolean(q && inquiry.error)} minutes={totalMinutes} />
        {/* 후속 답의 진행은 늘 있는 라이브 영역으로 읽어 준다(내용만 바뀜). */}
        <p role="status" aria-live="polite" className="sr-only">{followupAnnouncement}</p>
        {q && !generating && q.error && hasContent && (
          <p role="alert" className="mt-3 flex flex-wrap items-center gap-2 break-keep rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">
            {t.regenerateFailed}
            <Button size="sm" variant="ghost" onClick={() => q && regenerate.mutate(q.id)} loading={regenerate.isPending}>
              {t.retry}
            </Button>
          </p>
        )}
        {q ? (
          <InquiryHeader question={q.question} />
        ) : (
          <div aria-hidden="true" className="flex flex-col gap-3 pt-4">
            <Skeleton className="w-16" />
            <Skeleton lines={2} />
            <Skeleton className="w-2/3" />
          </div>
        )}

        <div aria-busy={!q || generating}>
          {hasContent && (
            <div ref={bodyRef} className="mt-6">
              <LectureBody content={{ sections }} onBeforeDeeper={() => undefined} sectionNote={sectionNote} />
            </div>
          )}
          {unanswered.map((f) => (
            <section key={f.id} aria-label={t.followupRegion} className="mt-10">
              <p className="mb-3 border-b border-line pb-2 text-xs font-medium text-fg-muted">{t.followupHeading}</p>
              <FollowupQuestion followup={f} />
              {f.status === "pending" ? (
                <div className="flex flex-col gap-3">
                  <p className="flex items-center gap-2 text-sm text-fg-secondary">
                    <ProgressCircle size="24" tone="inherit" aria-hidden />
                    {t.followupWriting}
                  </p>
                  <Skeleton lines={3} />
                </div>
              ) : (
                <div className="flex flex-wrap items-center gap-2 text-sm text-warning">
                  {t.followupFailed}
                  <Button size="sm" variant="secondary" loading={retryFollowup.isPending} onClick={() => retryFollowup.mutate(f.id)}>
                    {t.retry}
                  </Button>
                </div>
              )}
            </section>
          ))}
          {hasContent && !generating && followups.length < INQUIRY_MAX_FOLLOWUPS && <QuoteAsk containerRef={bodyRef} onQuote={pickQuote} />}
          {!hasContent && (!q || generating) && (
            <div className="mt-6 flex flex-col gap-6" aria-hidden="true">
              <Skeleton lines={4} />
              <Skeleton lines={3} className="h-40" />
              <Skeleton lines={5} />
            </div>
          )}
          {q && !hasContent && !generating && (
            <div className="mt-6">
              <ErrorState
                title={t.noContent}
                onRetry={() => q && regenerate.mutate(q.id)}
              />
            </div>
          )}

          {hasContent && q?.concepts && q.concepts.length > 0 && (
            <div className="mt-10 xl:hidden">
              <InquiryConcepts concepts={q.concepts} />
            </div>
          )}
        </div>

        {hasContent && (
          <div className="mt-10 flex flex-col gap-2 border-t border-line pt-4">
            {read && <p role="status" className="text-sm text-fg-secondary">{t.readDone}</p>}
            <div className="flex flex-wrap items-center gap-2">
              {read ? (
                <Button variant="ghost" onClick={toggleRead} loading={setRead.isPending} disabled={generating}>
                  {t.unread}
                </Button>
              ) : (
                <Button variant="primary" onClick={toggleRead} loading={setRead.isPending} disabled={generating}>
                  {t.markRead}
                </Button>
              )}
              {!generating && followups.length < INQUIRY_MAX_FOLLOWUPS && (
                <Button variant="secondary" onClick={askMore}>
                  {t.askMore}
                </Button>
              )}
            </div>
          </div>
        )}
        {hasContent && !generating && (
          <FollowupComposer
            ref={composerRef}
            inquiryId={inquiryId}
            used={followups.length}
            busy={followupBusy}
            quote={quote}
            onClearQuote={() => setQuote("")}
          />
        )}
        <div aria-live="polite" className="empty:hidden">
          <InlineError error={regenerate.error ?? setRead.error ?? retryFollowup.error ?? (q ? inquiry.error : null)} />
        </div>
      </LectureLayout>
      <InquiryTocSheet open={!wide && panel === "toc"} onClose={closePanel} sections={sections} totalMinutes={totalMinutes} />
    </div>
  );
}
