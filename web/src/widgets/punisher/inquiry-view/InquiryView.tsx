import { Link } from "@/src/shared/lib/router";
import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { AlertTriangle, CheckCircle2 } from "lucide-react";
import { BottomSheet, Button, ErrorState, Skeleton, Spinner, buttonVariants, type MenuItem } from "@/src/shared/ui/kit";
import { useInquiry, useRegenerateInquiry, useRetryFollowup, useSetInquiryRead } from "@/src/entities/punisher";
import { describePunisherError, type Inquiry, type InquiryFollowup, type LectureSection } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { LectureAssistPlaceholder, LectureBody, LectureLayout, LectureTopBar, TocSheet, useActiveSection } from "../lecture-viewer";
import { InquiryConcepts } from "./ui/InquiryConcepts";
import { FollowupComposer } from "./ui/FollowupComposer";
import { InquiryHeader } from "./ui/InquiryHeader";
import { QuoteAsk } from "./ui/QuoteAsk";
import { FollowupQuestion, inquirySectionNote } from "./ui/sectionNote";

const HOME = "/";
const NO_SECTIONS: LectureSection[] = [];
const NO_FOLLOWUPS: InquiryFollowup[] = [];
const dateFormat = new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium", timeStyle: "short" });

/**
 * 해설 상태 줄 — 항상 같은 자리·같은 높이(쓰는 중 ↔ 완료가 바뀌어도 본문이 밀리지 않음)이고, aria-busy 밖의
 * 라이브 영역이라 완료도 읽어 준다.
 */
function StatusRow({ inquiry, hasContent }: { inquiry: Inquiry | undefined; hasContent: boolean }) {
  let content: ReactNode = <Skeleton className="w-40" />;
  if (inquiry?.status === "pending") {
    content = (
      <>
        <Spinner size="sm" label={null} />
        {hasContent ? "새 해설을 쓰는 중이에요 · 끝나면 바뀌어요" : "해설을 쓰는 중이에요 · 보통 1~3분, 나가도 계속 써요"}
      </>
    );
  } else if (inquiry?.status === "failed") {
    content = (
      <>
        <AlertTriangle className="size-4 shrink-0 text-warning" aria-hidden="true" />
        해설을 만들지 못했어요
      </>
    );
  } else if (inquiry) {
    content = (
      <>
        <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />
        해설 완료{inquiry.answeredAt ? ` · ${dateFormat.format(new Date(inquiry.answeredAt))}` : ""}
        {inquiry.readAt ? " · 다 봄" : ""}
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
  const inquiry = useInquiry(inquiryId);
  const regenerate = useRegenerateInquiry();
  const setRead = useSetInquiryRead();
  const [tocOpen, setTocOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const retryFollowup = useRetryFollowup(inquiryId);
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
  const sectionNote = useCallback((section: LectureSection) => inquirySectionNote(section, followupByAnchor), [followupByAnchor]);
  const unanswered = followups.filter((f) => f.status !== "answered");
  const followupBusy = unanswered.some((f) => f.status === "pending");
  const pickQuote = useCallback((text: string) => {
    setQuote(text);
    composerRef.current?.focus();
    composerRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, []);
  const anchorKey = useMemo(() => sections.map((s) => s.anchor).filter(Boolean).join("|"), [sections]);
  const activeAnchor = useActiveSection(anchorKey);

  // 처음 불러오기에 실패했을 때만 오류 화면. 폴링 중 일시 오류는 본문을 지우지 않고 아래에 알린다.
  if (!q && inquiry.error) {
    return <ErrorState message={describePunisherError(inquiry.error)} onRetry={() => inquiry.refetch()} retrying={inquiry.isRefetching} />;
  }

  const hasContent = baseSections.length > 0;
  const generating = q?.status === "pending";
  const read = Boolean(q?.readAt);
  const totalMinutes = sections.reduce((sum, s) => sum + (s.minutes || 0), 0);
  const askRegenerate = () => (hasContent ? setConfirmOpen(true) : q && regenerate.mutate(q.id));
  const runRegenerate = () => {
    setConfirmOpen(false);
    if (q) { regenerate.mutate(q.id); }
  };
  const toggleRead = () => q && setRead.mutate({ id: q.id, read: !read });

  const menuItems: MenuItem[] = q
    ? [
        { key: "read", label: read ? "목록에 다시 두기" : "다 봤어요", disabled: generating || setRead.isPending, onSelect: toggleRead },
        { key: "regenerate", label: "다시 만들기", disabled: generating || regenerate.isPending, onSelect: askRegenerate },
      ]
    : [];

  let assist: ReactNode = <LectureAssistPlaceholder>해설이 끝나면 여기에 관련 개념과 지도에 새로 제안한 개념이 보여요.</LectureAssistPlaceholder>;
  if (q?.concepts && q.concepts.length > 0) { assist = <InquiryConcepts concepts={q.concepts} />; }
  else if (q && !generating) { assist = <LectureAssistPlaceholder>이 해설에는 지도와 이어진 개념이 없어요.</LectureAssistPlaceholder>; }

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar
        title={q?.content?.title || q?.question || "질문 해설"}
        backHref={HOME}
        onOpenToc={hasContent ? () => setTocOpen(true) : undefined}
        menuItems={menuItems}
      />
      <LectureLayout sections={sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} assist={assist}>
        <StatusRow inquiry={q} hasContent={hasContent} />
        {q && !generating && q.error && hasContent && (
          <p role="alert" className="mt-3 flex flex-wrap items-center gap-2 break-keep rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">
            다시 만들기에 실패해 이전 해설을 보여 드려요.
            <Button size="sm" variant="ghost" onClick={runRegenerate} loading={regenerate.isPending}>
              다시 시도
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
            <section key={f.id} aria-label="이어서 물은 질문" className="mt-10">
              <p className="mb-3 border-b border-line pb-2 text-xs font-medium text-fg-muted">이어서 묻기</p>
              <FollowupQuestion followup={f} />
              {f.status === "pending" ? (
                <div className="flex flex-col gap-3">
                  <p role="status" className="flex items-center gap-2 text-sm text-fg-secondary">
                    <Spinner size="sm" label={null} />
                    답을 쓰는 중이에요 · 보통 1분 안팎
                  </p>
                  <Skeleton lines={3} />
                </div>
              ) : (
                <div className="flex flex-wrap items-center gap-2 text-sm text-warning">
                  답을 만들지 못했어요.
                  <Button size="sm" variant="secondary" loading={retryFollowup.isPending} onClick={() => retryFollowup.mutate(f.id)}>
                    다시 시도
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
          {hasContent && <QuoteAsk containerRef={bodyRef} onQuote={pickQuote} />}
          {!hasContent && (!q || generating) && (
            <div className="mt-6 flex flex-col gap-6" aria-hidden="true">
              <Skeleton lines={4} />
              <Skeleton variant="block" className="h-40" />
              <Skeleton lines={5} />
            </div>
          )}
          {q && !hasContent && !generating && (
            <div className="mt-6">
              <ErrorState
                message="해설을 만들지 못했어요. 잠시 뒤 다시 만들어 보세요."
                onRetry={runRegenerate}
                retrying={regenerate.isPending}
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
              {read ? "목록에 다시 두기" : "다 봤어요"}
            </Button>
          )}
          {hasContent && (
            <Button variant="ghost" onClick={askRegenerate} loading={regenerate.isPending} disabled={generating}>
              다시 만들기
            </Button>
          )}
          <Link href={HOME} className={buttonVariants({ variant: "ghost" })}>
            학습 홈으로
          </Link>
        </div>
        {read && hasContent && <p className="mt-2 text-xs text-fg-muted">학습 홈 목록에서 빠졌어요. 지도의 관련 개념과 기록에는 남아요.</p>}
        <div aria-live="polite" className="empty:hidden">
          <InlineError error={regenerate.error ?? setRead.error ?? retryFollowup.error ?? (q ? inquiry.error : null)} />
        </div>
      </LectureLayout>
      {hasContent && <TocSheet open={tocOpen} onClose={() => setTocOpen(false)} sections={sections} totalMinutes={totalMinutes} />}
      <BottomSheet
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title="새 해설로 다시 만들까요?"
        description="같은 질문으로 새로 써요(1~3분)"
        footer={
          <>
            <Button variant="ghost" onClick={() => setConfirmOpen(false)}>
              취소
            </Button>
            <Button onClick={runRegenerate} loading={regenerate.isPending}>
              다시 만들기
            </Button>
          </>
        }
      >
        <ul className="list-disc space-y-1 break-keep pl-5 text-sm text-fg-secondary">
          <li>새 해설이 완성되면 지금 해설을 대신해요. 그동안은 지금 해설을 그대로 볼 수 있어요.</li>
          <li>만들기에 실패하면 지금 해설이 남아요.</li>
          <li>생성 사용량이 한 번 더 들어요.</li>
        </ul>
      </BottomSheet>
    </div>
  );
}
