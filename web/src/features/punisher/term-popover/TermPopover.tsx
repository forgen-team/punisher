
import { useState } from "react";
import { BookOpen } from "lucide-react";
import { Button, Skeleton, cn, focusRing } from "@/src/shared/ui/kit";
import { useGloss } from "@/src/entities/punisher";
import { InlineError, ReadingOverlay, toOverlayAnchor, type OverlayAnchor } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useLaunchInstant } from "../instant-start";

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
  const [anchor, setAnchor] = useState<OverlayAnchor | null>(null);
  const open = anchor !== null;
  const gloss = useGloss(conceptId, open);
  const { launch, isPending, error } = useLaunchInstant();
  const label = term || conceptId;

  async function goDeeper() {
    const origin = onBeforeDeeper();
    try {
      await launch(origin ? { conceptId, parentLectureId: origin.lectureId, parentAnchor: origin.anchor } : { conceptId });
    } catch {
      // 시트를 유지하고 오류를 아래에 표시한다.
      return;
    }
  }

  const text = gloss.data?.gloss || fallbackText || "";

  return (
    <>
      <button
        type="button"
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={(event) => setAnchor(toOverlayAnchor(event.currentTarget.getBoundingClientRect()))}
        className={cn(
          inline
            ? "inline rounded-sm text-accent-text underline decoration-dotted underline-offset-4"
            : "inline-flex min-h-touch items-center gap-1.5 rounded-sm text-base font-medium text-accent-text underline decoration-dotted underline-offset-4",
          focusRing,
        )}
      >
        {!inline && <BookOpen className="size-4" aria-hidden="true" />}
        {label}
        {proposed && <span className="text-xs font-normal text-fg-muted no-underline">(제안)</span>}
      </button>
      <ReadingOverlay
        open={open}
        onClose={() => setAnchor(null)}
        anchor={anchor}
        title={label}
        footer={
          <Button size="sm" variant="secondary" loading={isPending} onClick={goDeeper}>
            이 개념 더 깊게
          </Button>
        }
      >
        {gloss.isLoading && !fallbackText ? (
          <Skeleton lines={3} />
        ) : text ? (
          <Markdown className="text-base leading-[1.7]">{text}</Markdown>
        ) : (
          <InlineError error={gloss.error} />
        )}
        <div aria-live="polite" className="mt-2">
          <InlineError error={error} />
        </div>
      </ReadingOverlay>
    </>
  );
}
