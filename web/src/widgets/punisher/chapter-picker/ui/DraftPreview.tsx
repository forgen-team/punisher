
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

/** 미리보기: 배우는 순서대로 개념·선행·목표 수 — '이대로 열기'를 눌러야 지도에 반영된다. */
export function DraftPreview({ draft, names, applying, onApply, onBack }: DraftPreviewProps) {
  const inChapter = new Map(draft.concepts.map((c) => [c.id, c.name]));
  const nameOf = (id: string) => inChapter.get(id) ?? names.get(id) ?? id;
  const newCount = draft.concepts.filter((c) => c.action !== "reuse").length;

  return (
    <Section
      variant="card"
      title={`새 챕터: ${draft.title}`}
      description={`${draft.summary} · 새로 열리는 개념 ${newCount}개 · 내 지도에만 보여요`}
    >
      <ol className="flex flex-col gap-3">
        {draft.concepts.map((c, index) => (
          <li key={c.id} className="flex gap-3 border-b border-line pb-3 last:border-b-0">
            <span className="mt-0.5 w-6 shrink-0 text-right text-sm font-semibold text-fg-muted">{index + 1}</span>
            <div className="min-w-0 flex-1">
              <p className="flex flex-wrap items-center gap-1.5 text-base font-semibold text-fg">
                {c.name}
                <Badge tone={c.action === "create" ? "accent" : "neutral"}>{CONCEPT_ACTION_LABELS[c.action]}</Badge>
              </p>
              <p className="text-xs text-fg-muted">
                {areaLabel(c.area)} · 깊이 {c.depth}
                {c.action !== "reuse" && ` · 목표 ${c.objectives?.length ?? 0}개`}
              </p>
              {c.summary && c.action !== "reuse" && <p className="mt-1 text-sm text-fg-secondary">{c.summary}</p>}
              {(c.prerequisites ?? []).length > 0 && (
                <p className="mt-1 text-xs text-fg-secondary">먼저 알면 좋은 것: {(c.prerequisites ?? []).map((id) => nameOf(id)).join(", ")}</p>
              )}
            </div>
          </li>
        ))}
      </ol>
      <div className="mt-4 flex flex-wrap justify-end gap-2">
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
