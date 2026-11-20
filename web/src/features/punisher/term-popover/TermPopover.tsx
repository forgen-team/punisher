
import { useState } from "react";
import { BookOpen } from "lucide-react";
import { Button, Chip, Skeleton, cn, focusRing } from "@/src/shared/ui/kit";
import { useCreateStudyRequest, useGloss } from "@/src/entities/punisher";
import { SELECTION_MIN_CHARS, type ThreadMode } from "@/src/shared/api/punisher";
import { InlineError, ReadingOverlay, toOverlayAnchor, type OverlayAnchor } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useLaunchInstant } from "../instant-start";
import { useReadingThreads } from "../reading-assist/ReadingThreadsProvider";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 사전 용어 id(term:g.…) — 지도 개념이 아니다. */
export function isGlossaryTerm(id: string): boolean {
  return id.startsWith("g.");
}

/** "더 깊게"를 누른 원 강의와 위치. */
export interface DeeperOrigin {
  lectureId: string;
  anchor: string;
}

interface TermPopoverProps {
  term: string;
  conceptId: string;
  fallbackText?: string;
  proposed?: boolean;
  /**
   * Called before navigating to the deeper lecture (saves the current anchor).
   * Returns the origin so the new lecture is linked under it ("이 강의에서 더 깊게 본 강의").
   */
  onBeforeDeeper: () => DeeperOrigin | void;
  /** 본문 문장 안의 인라인 링크(아이콘·터치 높이 없이 글자만) */
  inline?: boolean;
}

/**
 * 강의 본문의 개념 용어 — 누르면 짧은 설명 오버레이(모바일 하단 시트, 데스크톱 용어 옆 팝오버).
 * 본문에 펼치지 않고 스크롤 위치도 바꾸지 않는다(스크롤 잠금 없음).
 * Esc·바깥·닫기로 본문 복귀, "더 깊게"는 현재 위치를 북마크하고 해당 개념 강의로 이동 (design §3.3).
 */
export function TermPopover({ term, conceptId, fallbackText, proposed, onBeforeDeeper, inline = false }: TermPopoverProps) {
  const t = useT(messages);
  const [anchor, setAnchor] = useState<OverlayAnchor | null>(null);
  // 용어가 놓인 본문 블록의 진도 앵커 — 용어로 스레드를 시작할 때 구절 위치로 쓴다.
  const [blockAnchor, setBlockAnchor] = useState("");
  const open = anchor !== null;
  const threads = useReadingThreads();
  const readLater = useCreateStudyRequest();
  const gloss = useGloss(conceptId, open);
  const { launch, isPending, error } = useLaunchInstant();
  const label = term || conceptId;

  // 사전 용어(g.…)는 그 자체로 강의가 없다 — 가장 가까운 지도 개념으로 '더 깊게'(없으면 버튼 없음).
  const deeperId = isGlossaryTerm(conceptId) ? (gloss.data?.relatedConceptId ?? "") : conceptId;

  async function goDeeper() {
    const origin = onBeforeDeeper();
    try {
      await launch(origin ? { conceptId: deeperId, parentLectureId: origin.lectureId, parentAnchor: origin.anchor } : { conceptId: deeperId });
    } catch {
      // 시트를 유지하고 오류를 아래에 표시한다.
      return;
    }
  }

  const text = gloss.data?.gloss || fallbackText || "";

  // 서버는 SELECTION_MIN_CHARS 보다 짧은 구절을 받지 않는다(한 글자 용어 등) — 그때는 스레드 칩을 숨긴다.
  const canThread = threads !== null && Array.from(label.trim()).length >= SELECTION_MIN_CHARS;

  /** 더 쉽게(바로 답) · 이어서 묻기(입력 열기) — 용어를 구절로 스레드를 시작하고 이 시트는 닫는다. */
  function startThread(mode: ThreadMode) {
    if (!threads) { return; }
    const rect = anchor;
    const passage = { anchor: blockAnchor || threads.getAnchor(), quote: label, rect };
    setAnchor(null);
    // 이 시트가 닫히는 프레임에 스레드 시트를 열면 두 오버레이의 포커스 복귀·닫힘 애니메이션이 엉킨다 — 다음 프레임에 연다.
    requestAnimationFrame(() => threads.start(passage, mode));
  }

  return (
    <>
      <button
        type="button"
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={(event) => {
          setBlockAnchor(event.currentTarget.closest<HTMLElement>("[data-lecture-anchor]")?.dataset.lectureAnchor ?? "");
          setAnchor(toOverlayAnchor(event.currentTarget.getBoundingClientRect()));
        }}
        className={cn(
          // 문장 속 용어: 보이는 크기·행간은 그대로, 위아래 터치 영역만 44px 이상(줄 높이 약 28px + 위아래 10px).
          inline
            ? "relative inline rounded-sm text-accent-text underline decoration-dotted underline-offset-4 before:absolute before:inset-x-0 before:-inset-y-2.5 before:content-['']"
            : "inline-flex min-h-touch items-center gap-1.5 rounded-sm text-base font-medium text-accent-text underline decoration-dotted underline-offset-4",
          focusRing,
        )}
      >
        {!inline && <BookOpen className="size-4" aria-hidden="true" />}
        {label}
        {proposed && <span className="text-xs font-normal text-fg-muted no-underline">{t.proposed}</span>}
      </button>
      <ReadingOverlay
        open={open}
        onClose={() => setAnchor(null)}
        anchor={anchor}
        title={label}
        footer={
          <>
            <Button
              size="sm"
              variant="ghost"
              loading={readLater.isPending}
              disabled={readLater.isSuccess}
              onClick={() => readLater.mutate(label)}
            >
              {readLater.isSuccess ? t.readLaterDone : t.readLater}
            </Button>
            {deeperId && (
              <Button size="sm" variant="secondary" loading={isPending} onClick={goDeeper}>
                {t.goDeeper}
              </Button>
            )}
          </>
        }
      >
        {gloss.isLoading && !fallbackText ? (
          <Skeleton lines={3} />
        ) : text ? (
          <Markdown className="text-base leading-[1.7]">{text}</Markdown>
        ) : (
          <InlineError error={gloss.error} />
        )}
        {canThread && text && (
          <div className="mt-3 flex flex-wrap gap-2">
            <Chip onClick={() => startThread("easier")}>{t.easier}</Chip>
            <Chip onClick={() => startThread("question")}>{t.askMore}</Chip>
          </div>
        )}
        <div aria-live="polite" className="mt-2">
          <InlineError error={error ?? readLater.error} />
        </div>
      </ReadingOverlay>
    </>
  );
}
