
import { useId } from "react";
import { Badge, Button, Section } from "@/src/shared/ui/kit";
import { areaLabel } from "@/src/entities/punisher";
import type { ChapterDraft } from "@/src/shared/api/punisher";
import { CONCEPT_ACTION_LABELS } from "./constants";

interface DraftPreviewProps {
  draft: ChapterDraft;
  /** 지도에 이미 있는 개념 id → 이름(선행 개념 표시용) */
  names: ReadonlyMap<string, string>;
  applying: boolean;
  onApply: () => void;
  onBack: () => void;
}

/** 영역 · 깊이 · 목표 수 — 지도에 이미 있는 개념(reuse)은 초안에 영역·깊이가 비어 올 수 있어 빈 값은 뺀다. */
function metaOf(c: ChapterDraft["concepts"][number]): string {
  return [
    c.area ? areaLabel(c.area) : "",
    c.depth > 0 ? `깊이 ${c.depth}` : "",
    c.action !== "reuse" ? `목표 ${c.objectives?.length ?? 0}개` : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** 미리보기: 배우는 순서대로 개념·선행·목표 수 — '이대로 열기'를 눌러야 지도에 반영된다. */
export function DraftPreview({ draft, names, applying, onApply, onBack }: DraftPreviewProps) {
  const inChapter = new Map(draft.concepts.map((c) => [c.id, c.name]));
  const nameOf = (id: string) => inChapter.get(id) ?? names.get(id) ?? id;
  const newCount = draft.concepts.filter((c) => c.action !== "reuse").length;
  const titleId = useId();

  return (
    <Section variant="card" aria-labelledby={titleId}>
      <header className="flex flex-col gap-1 border-b border-line py-4">
        <p className="text-xs font-semibold text-accent-text">새 챕터 미리보기</p>
        <h2 id={titleId} className="break-keep text-lg font-semibold text-fg">
          {draft.title}
        </h2>
        {draft.summary && <p className="break-keep text-sm text-fg-secondary">{draft.summary}</p>}
        <p className="text-xs text-fg-muted">새로 열리는 개념 {newCount}개 · 내 지도에만 보여요</p>
      </header>
      {/* xl: 두 단(위→아래로 이어지는 신문 단) — 배우는 순서를 위에서 아래로 그대로 읽는다. */}
      <ol className="xl:columns-2 xl:gap-x-8">
        {draft.concepts.map((c, index) => (
          <li key={c.id} className="flex break-inside-avoid gap-3 border-b border-line py-3 last:border-b-0">
            <span className="mt-0.5 w-6 shrink-0 text-right text-sm font-semibold tabular-nums text-fg-muted">{index + 1}</span>
            <div className="min-w-0 flex-1">
              <p className="flex flex-wrap items-center gap-1.5 text-base font-semibold text-fg">
                {c.name}
                <Badge tone={c.action === "create" ? "accent" : "neutral"}>{CONCEPT_ACTION_LABELS[c.action]}</Badge>
              </p>
              {metaOf(c) && <p className="mt-0.5 text-xs text-fg-muted">{metaOf(c)}</p>}
              {c.summary && c.action !== "reuse" && <p className="mt-1 break-keep text-sm text-fg-secondary">{c.summary}</p>}
              {(c.prerequisites ?? []).length > 0 && (
                <p className="mt-1 break-keep text-xs text-fg-secondary">먼저 알면 좋은 것: {(c.prerequisites ?? []).map((id) => nameOf(id)).join(", ")}</p>
              )}
            </div>
          </li>
        ))}
      </ol>
      <div className="flex flex-wrap justify-end gap-2 border-t border-line py-4">
        <Button variant="ghost" onClick={onBack} disabled={applying}>
          다시 고르기
        </Button>
        <Button onClick={onApply} loading={applying}>
          이대로 열기
        </Button>
      </div>
    </Section>
  );
}
