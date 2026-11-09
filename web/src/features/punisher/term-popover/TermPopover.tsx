
import { useState } from "react";
import { BookOpen } from "lucide-react";
import { Button, Skeleton, cn, focusRing } from "@/src/shared/ui/kit";
import { useGloss } from "@/src/entities/punisher";
import { InlineError, ReadingOverlay, toOverlayAnchor, type OverlayAnchor } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useLaunchInstant } from "../instant-start";

interface TermPopoverProps {
  term: string;
  conceptId: string;
  fallbackText?: string;
  proposed?: boolean;
  /** Called before navigating to the deeper lecture (saves the current anchor). */
  onBeforeDeeper: () => void;
}

/**
 * 강의 본문의 개념 용어 — 누르면 짧은 설명 오버레이(모바일 하단 시트, 데스크톱 용어 옆 팝오버).
 * 본문에 펼치지 않고 스크롤 위치도 바꾸지 않는다(스크롤 잠금 없음).
 * Esc·바깥·닫기로 본문 복귀, "더 깊게"는 현재 위치를 북마크하고 해당 개념 강의로 이동 (design §3.3).
 */
export function TermPopover({ term, conceptId, fallbackText, proposed, onBeforeDeeper }: TermPopoverProps) {
  const [anchor, setAnchor] = useState<OverlayAnchor | null>(null);
  const open = anchor !== null;
  const gloss = useGloss(conceptId, open);
  const { launch, isPending, error } = useLaunchInstant();
  const label = term || conceptId;

  async function goDeeper() {
    onBeforeDeeper();
    try {
      await launch({ conceptId });
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
          "inline-flex min-h-touch items-center gap-1.5 rounded-sm text-base font-medium text-accent-text",
          "underline decoration-dotted underline-offset-4",
          focusRing,
        )}
      >
        <BookOpen className="size-4" aria-hidden="true" />
        {label}
        {proposed && <span className="text-xs font-normal text-fg-muted no-underline">(제안)</span>}
      </button>
      <ReadingOverlay
        open={open}
        onClose={() => setAnchor(null)}
        anchor={anchor}
        title={label}
        description="용어 설명 · 닫아도 읽던 위치 그대로예요"
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
