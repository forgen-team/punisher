import { Link } from "@/src/shared/lib/router";
import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { AlertTriangle, CheckCircle2 } from "lucide-react";
import { Button, ErrorState, ProgressCircle, Skeleton, useConfirm, type MenuItemDef } from "@forgen-team/ui";
import { useScreenTitle } from "@/src/shared/lib/screen-chrome";
import { useInquiry, useRegenerateInquiry, useRetryFollowup, useSetInquiryRead } from "@/src/entities/punisher";
import { INQUIRY_MAX_FOLLOWUPS, describePunisherError, type Inquiry, type InquiryFollowup, type LectureSection } from "@/src/shared/api/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { FOLLOWUP_ANNOUNCEMENTS, messages } from "./i18n";
import { LectureAssistPlaceholder, LectureBody, LectureLayout, LectureTopBar, TocSheet, useActiveSection } from "../lecture-viewer";
import { InquiryConcepts } from "./ui/InquiryConcepts";
import { FollowupComposer } from "./ui/FollowupComposer";
import { InquiryHeader } from "./ui/InquiryHeader";
import { QuoteAsk } from "./ui/QuoteAsk";
import { FollowupQuestion, inquirySectionNote } from "./ui/sectionNote";

const HOME = "/";
const NO_SECTIONS: LectureSection[] = [];
const NO_FOLLOWUPS: InquiryFollowup[] = [];
const ANSWERED_AT: Intl.DateTimeFormatOptions = { dateStyle: "medium", timeStyle: "short" };

/**
 * 해설 상태 줄 — 항상 같은 자리·같은 높이(쓰는 중 ↔ 완료가 바뀌어도 본문이 밀리지 않음)이고, aria-busy 밖의
 * 라이브 영역이라 완료도 읽어 준다.
 */
function StatusRow({ inquiry, hasContent, unstable }: { inquiry: Inquiry | undefined; hasContent: boolean; unstable: boolean }) {
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
    content = (
      <>
        <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />
        {t.done(inquiry.answeredAt ? formatDate(new Date(inquiry.answeredAt), locale, ANSWERED_AT) : "")}
        {inquiry.readAt ? t.readSuffix : ""}
      </>
    );
  }
  return (
    <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
      {content}
    </p>
  );
}

/**
 * '이 질문에 답하기' 해설 화면 — 강의와 같은 배치(상단 바 · 접은/펼친 목차 · 가독 폭 본문 · xl 보조 열).
 * 보조 열에는 해설에 나온 개념(지도에 있던 것 · 새로 제안한 것). 쓰는 중이면 3초마다 다시 확인하고,
 * 끝난 해설은 저장된 그대로 보인다('다시 만들기'에서만 새로 쓴다). 불러오는 중에도 같은 뼈대를 그려 밀림이 없다.
 */
export function InquiryView({ inquiryId }: { inquiryId: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const inquiry = useInquiry(inquiryId);
  const regenerate = useRegenerateInquiry();
  const setRead = useSetInquiryRead();
  const [tocOpen, setTocOpen] = useState(false);
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

  // 처음 불러오기에 실패했을 때만 오류 화면. 폴링 중 일시 오류는 본문을 지우지 않고 아래에 알린다.
  if (!q && inquiry.error) {
    return <ErrorState title={describePunisherError(inquiry.error)} onRetry={() => void inquiry.refetch()} />;
  }

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

  const menuItems: MenuItemDef[] = q
    ? [
        { key: "read", label: read ? t.unread : t.markRead, disabled: generating || setRead.isPending, onSelect: toggleRead },
        { key: "regenerate", label: t.regenerate, disabled: generating || followupBusy || regenerate.isPending, onSelect: () => void askRegenerate() },
      ]
    : [];

  let assist: ReactNode = <LectureAssistPlaceholder>{t.assistPending}</LectureAssistPlaceholder>;
  if (q?.concepts && q.concepts.length > 0) { assist = <InquiryConcepts concepts={q.concepts} />; }
  else if (q && !generating) { assist = <LectureAssistPlaceholder>{t.assistEmpty}</LectureAssistPlaceholder>; }

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar
        title={q?.content?.title || q?.question || t.title}
        backHref={HOME}
        onOpenToc={hasContent ? () => setTocOpen(true) : undefined}
        menuItems={menuItems}
        menuText={{ full: t.menuManage, short: t.menuManageShort }}
      />
      <LectureLayout sections={sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} assist={assist}>
        <StatusRow inquiry={q} hasContent={hasContent} unstable={Boolean(q && inquiry.error)} />
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
          <InquiryHeader question={q.question} title={q.content?.title} />
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

        <div className="mt-10 flex flex-wrap items-center gap-2 border-t border-line pt-4">
          {hasContent && (
            <Button onClick={toggleRead} loading={setRead.isPending} disabled={generating} variant={read ? "secondary" : "primary"}>
              {read ? t.unread : t.markRead}
            </Button>
          )}
          {hasContent && (
            <Button
              variant="ghost"
              onClick={() => void askRegenerate()}
              loading={regenerate.isPending}
              disabled={generating || followupBusy}
              title={followupBusy ? t.regenerateBusy : undefined}
            >
              {t.regenerate}
            </Button>
          )}
          <Button asChild variant="ghost">
            <Link href={HOME}>
              {t.home}
            </Link>
          </Button>
        </div>
        {read && hasContent && <p className="mt-2 text-xs text-fg-muted">{t.readNotice}</p>}
        <div aria-live="polite" className="empty:hidden">
          <InlineError error={regenerate.error ?? setRead.error ?? retryFollowup.error ?? (q ? inquiry.error : null)} />
        </div>
      </LectureLayout>
      {hasContent && <TocSheet open={tocOpen} onClose={() => setTocOpen(false)} sections={sections} totalMinutes={totalMinutes} />}
    </div>
  );
}
