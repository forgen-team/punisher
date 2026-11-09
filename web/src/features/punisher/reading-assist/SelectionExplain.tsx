
import { useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { Sparkles } from "lucide-react";
import { Button, Skeleton, cn, focusRing, useMediaQuery } from "@/src/shared/ui/kit";
import { useExplainSelection } from "@/src/entities/punisher";
import { SELECTION_MAX_CHARS, SELECTION_MIN_CHARS } from "@/src/shared/api/punisher";
import { InlineError, ReadingOverlay } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useTextSelection, type TextSelectionSnapshot } from "./useTextSelection";

const BUTTON_GAP = 10;

function ExplanationBody({ pending, text, error }: { pending: boolean; text?: string; error: unknown }) {
  if (pending) {
    return (
      <div aria-busy="true" aria-label="설명을 만드는 중">
        <Skeleton lines={4} />
      </div>
    );
  }
  if (text) { return <Markdown className="text-base leading-[1.7]">{text}</Markdown>; }
  return <InlineError error={error} />;
}

interface SelectionExplainProps {
  lectureId: string;
  /** 강의 본문 컨테이너 — 이 안의 선택에만 반응한다 */
  containerRef: RefObject<HTMLElement>;
}

/**
 * 본문에서 드래그/길게 눌러 고른 구절 옆에 "이 부분 설명" 버튼을 띄우고,
 * 누르면 같은 읽기 오버레이(데스크톱 팝오버·모바일 시트)에 섹션 맥락 설명을 보여 준다.
 * 모바일은 OS 선택 메뉴가 위에 뜨므로 버튼을 선택 아래에 둔다.
 */
export function SelectionExplain({ lectureId, containerRef }: SelectionExplainProps) {
  const coarse = useMediaQuery("(pointer: coarse)");
  const [target, setTarget] = useState<TextSelectionSnapshot | null>(null);
  const selection = useTextSelection(containerRef, target === null);
  const explain = useExplainSelection(lectureId);

  function run(snapshot: TextSelectionSnapshot) {
    setTarget(snapshot);
    explain.reset();
    explain.mutate({ anchor: snapshot.anchor, selection: snapshot.text });
  }

  const length = selection ? Array.from(selection.text).length : 0;
  const showButton = selection !== null && length >= SELECTION_MIN_CHARS;
  const tooLong = length > SELECTION_MAX_CHARS;

  return (
    <>
      {showButton &&
        createPortal(
          <div
            className="absolute z-panel"
            style={{
              top: (coarse ? selection.rect.bottom + BUTTON_GAP : selection.rect.top - BUTTON_GAP - 40) + window.scrollY,
              left: Math.max(8, Math.min(selection.rect.left, window.innerWidth - 180)) + window.scrollX,
            }}
          >
            <button
              type="button"
              // 마우스 버튼을 누르는 순간 선택이 풀리지 않게 한다(데스크톱).
              onPointerDown={(event) => event.preventDefault()}
              onClick={() => run(selection)}
              disabled={tooLong}
              title={tooLong ? `${SELECTION_MAX_CHARS}자 이하로 골라 주세요` : undefined}
              className={cn(
                "inline-flex h-10 items-center gap-1.5 rounded-full border border-line bg-canvas-elevated px-3.5 text-sm font-medium",
                "text-fg shadow-overlay hover:bg-canvas-hover disabled:opacity-60",
                focusRing,
              )}
            >
              <Sparkles className="size-4 text-accent-text" aria-hidden="true" />
              {tooLong ? `너무 길어요(최대 ${SELECTION_MAX_CHARS}자)` : "이 부분 설명"}
            </button>
          </div>,
          document.body,
        )}
      <ReadingOverlay
        open={target !== null}
        onClose={() => setTarget(null)}
        anchor={target?.rect ?? null}
        title="이 부분 설명"
        description={explain.data?.sectionTitle ? `섹션: ${explain.data.sectionTitle}` : "고른 구절을 이 강의 맥락으로 풀어 드려요"}
        footer={
          explain.isError && target ? (
            <Button size="sm" variant="secondary" onClick={() => run(target)}>
              다시 시도
            </Button>
          ) : undefined
        }
      >
        {target && (
          <blockquote className="mb-3 line-clamp-4 border-l-2 border-line-strong pl-3 text-sm text-fg-secondary">
            {target.text}
          </blockquote>
        )}
        <ExplanationBody pending={explain.isPending} text={explain.data?.explanation} error={explain.error} />
      </ReadingOverlay>
    </>
  );
}
