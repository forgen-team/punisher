import { useRouter } from "@/src/shared/lib/router";
import { useMemo, useState, type ReactNode } from "react";
import { Button, ErrorState, Section, TextField } from "@forgen-team/ui";
import { describeCheckError, useApplyChapterDraft, useChapterSuggestions, useCreateChapterDraft, useGraph, useStartDiagnostic } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { PunisherApiError, describePunisherError, type ChapterDraftResponse } from "@/src/shared/api/punisher";
import { useDraftState } from "@/src/shared/lib/drafts";
import { DraftPreview } from "./ui/DraftPreview";
import { SuggestionCards, SuggestionCardsSkeleton } from "./ui/SuggestionCards";
import { FREE_TEXT_MAX, MAX_PICKS } from "./ui/constants";
import { useLocale, translate } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { pickSummary } from "./lib/pickSummary";

/** 422 CHAPTER_INVALID 이면 서버가 준 문제 목록을 함께 보여 준다. */
function designProblems(error: unknown): string[] {
  return error instanceof PunisherApiError && error.status === 422 ? error.problems : [];
}

interface PickerDraft {
  selected: string[];
  freeText: string;
  /** 설계해 본 초안(미리보기) — 다른 화면에 갔다 와도 그대로 본다 */
  preview: ChapterDraftResponse | null;
}

const EMPTY_DRAFT: PickerDraft = { selected: [], freeText: "", preview: null };

/** 단계 줄 "n/4 · 단계 이름" + 진행 막대. */
function StepMeta({ step, total, label, progress }: { step: number; total: number; label: string; progress: string }) {
  const percent = Math.round((step / total) * 100);
  return (
    <div>
      <p aria-live="polite" className="text-sm text-fg-secondary tabular-nums">{label}</p>
      <div
        role="progressbar"
        aria-label={progress}
        aria-valuenow={percent}
        aria-valuemin={0}
        aria-valuemax={100}
        className="mt-2 h-1 w-full overflow-hidden rounded-full bg-canvas-hover"
      >
        <div className="h-full bg-accent-solid" style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

/** 고정 Footer — 하단 탭 위(--app-bottom-offset). 왼쪽 상태·보조, 오른쪽 주 동작. */
function Footer({ children }: { children: ReactNode }) {
  return (
    <div className="sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 flex items-center gap-2 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0">
      {children}
    </div>
  );
}

/** 지도에 반영한 챕터 — 배치 확인 단계와 끝 화면이 이어서 쓴다. */
interface OpenedChapter {
  chapterId: string;
  title: string;
  conceptCount: number;
}

/**
 * 새 챕터(T-steps) — 제안 고르기 → 설계 → 반영 → 배치 확인(n/4 · 단계 이름, Footer 고정). 제목·←·설정은 틀이 그린다.
 * 고른 제안·직접 입력·설계 미리보기는 이 기기에 보관한다(점검 17) — 반영하면 지운다. 생성은 Claude 게이트웨이로 하고,
 * 반영 전까지 지도는 바뀌지 않는다. 배치 확인은 /check/:id(배치 확인 세션)로 이어지고, 건너뛰면 "열었어요" 끝 화면.
 */
export function ChapterPickerWidget() {
  const router = useRouter();
  const locale = useLocale();
  const t = translate(messages, locale);
  const suggestions = useChapterSuggestions();
  const graph = useGraph();
  const create = useCreateChapterDraft();
  const apply = useApplyChapterDraft();
  const draft = useDraftState<PickerDraft>("chapter-picker", EMPTY_DRAFT, "", (v) => v.selected.length === 0 && !v.freeText.trim() && !v.preview);
  const setDraft = draft.setValue;
  const selected = useMemo(() => new Set(draft.value.selected), [draft.value.selected]);
  const freeText = draft.value.freeText;
  const setFreeText = (text: string) => setDraft((v) => ({ ...v, freeText: text }));
  const preview = create.data ?? draft.value.preview;
  const [opened, setOpened] = useState<OpenedChapter | null>(null);
  const [finished, setFinished] = useState(false);
  const placement = useStartDiagnostic();

  const names = useMemo(() => new Map((graph.data?.nodes ?? []).map((n) => [n.id, n.name])), [graph.data]);

  function toggle(id: string) {
    setDraft((v) => {
      const next = new Set(v.selected);
      if (next.has(id)) { next.delete(id); } else if (next.size < MAX_PICKS) { next.add(id); }
      return { ...v, selected: [...next] };
    });
  }

  function design() {
    create.mutate(
      { suggestionIds: [...selected], freeText: freeText.trim() },
      { onSuccess: (res) => setDraft((v) => ({ ...v, preview: res })) },
    );
  }

  function openChapter() {
    if (!preview) { return; }
    const title = preview.draft.title;
    const conceptCount = preview.draft.concepts.filter((c) => c.action !== "reuse").length;
    apply.mutate(preview.draftId, {
      onSuccess: (res) => {
        draft.discard();
        setOpened({ chapterId: res.chapterId, title, conceptCount });
      },
    });
  }

  function goToMap(chapterId: string) {
    router.replace(`/map?chapter=${encodeURIComponent(chapterId)}`);
  }

  function startPlacement(chapterId: string) {
    placement.mutate(
      { kind: "placement", chapterId },
      { onSuccess: (detail) => { if (detail) { router.replace(checkHref(detail.session.id)); } } },
    );
  }

  const stepNames = [t.stepPick, t.stepDesign, t.stepApply, t.stepPlacement];
  const meta = (step: number) => (
    <StepMeta step={step} total={stepNames.length} label={t.stepMeta(step + 1, stepNames[step] ?? "")} progress={t.progress} />
  );

  // 끝 화면 — 무엇이 됐는지 + 다음 행동 1개.
  if (opened && finished) {
    return (
      <div className="flex flex-col items-start gap-2 pt-4">
        <p className="text-xl font-semibold text-fg">{t.openedTitle(opened.title)}</p>
        <Button variant="primary" className="mt-3" onClick={() => goToMap(opened.chapterId)}>
          {t.toMap}
        </Button>
      </div>
    );
  }

  // 3/4 반영 — 지도에 열렸다. 배치 확인으로 이어가거나 건너뛴다.
  if (opened) {
    return (
      <div className="flex flex-col gap-4 pt-4">
        {meta(2)}
        <h2 className="text-xl font-semibold text-fg">{t.appliedTitle}</h2>
        <p className="text-sm text-fg-secondary">{t.appliedCount(opened.conceptCount)}</p>
        {Boolean(placement.error) && (
          <p role="alert" className="text-sm text-danger">
            {describeCheckError(placement.error)}
          </p>
        )}
        <Footer>
          <Button variant="ghost" onClick={() => setFinished(true)} disabled={placement.isPending}>
            {t.placementSkip}
          </Button>
          <span className="flex-1" />
          <Button variant="primary" onClick={() => startPlacement(opened.chapterId)} loading={placement.isPending}>
            {t.placementStart}
          </Button>
        </Footer>
      </div>
    );
  }

  // 2/4 설계 — 미리보기를 보고 지도에 반영한다.
  if (preview) {
    return (
      <div className="flex flex-col gap-4 pt-4">
        {meta(1)}
        <DraftPreview draft={preview.draft} names={names} />
        {apply.error && (
          <p role="alert" className="text-sm text-danger">
            {describePunisherError(apply.error)}
          </p>
        )}
        <Footer>
          <Button
            variant="ghost"
            disabled={apply.isPending}
            onClick={() => {
              create.reset();
              apply.reset();
              setDraft((v) => ({ ...v, preview: null }));
            }}
          >
            {t.back}
          </Button>
          <span className="flex-1" />
          <Button variant="primary" onClick={openChapter} loading={apply.isPending}>
            {t.apply}
          </Button>
        </Footer>
      </div>
    );
  }

  // 1/4 제안 고르기
  const canDesign = (selected.size > 0 || freeText.trim() !== "") && !create.isPending;
  const problems = designProblems(create.error);

  return (
    <div className="flex flex-col gap-6 pt-4">
      {meta(0)}
      <h2 className="text-xl font-semibold text-fg">{suggestions.data?.question ?? t.defaultQuestion}</h2>

      <Section title={t.suggestions} description={t.maxPicks(MAX_PICKS)}>
        {suggestions.isLoading ? (
          <SuggestionCardsSkeleton />
        ) : suggestions.error ? (
          <ErrorState title={describePunisherError(suggestions.error)} onRetry={() => suggestions.refetch()} />
        ) : (
          <SuggestionCards suggestions={suggestions.data?.suggestions ?? []} selected={selected} onToggle={toggle} />
        )}
      </Section>

      <Section title={t.freeText} description={t.charCount(freeText.length, FREE_TEXT_MAX)}>
        <div className="flex flex-col gap-4">
          <TextField
            multiline
            value={freeText}
            onValueChange={({ value }) => setFreeText(value)}
            placeholder={t.freeTextPlaceholder}
            hideCharacterCount
            textareaProps={{ "aria-label": t.freeText, maxLength: FREE_TEXT_MAX, rows: 3 }}
          />
          <div aria-live="polite" className="empty:hidden">
            {create.error && (
              <div role="alert" className="bg-danger-subtle px-3 py-2 text-sm text-danger">
                <p>{describePunisherError(create.error)}</p>
                {problems.length > 0 && (
                  <ul className="mt-1 list-disc pl-4 text-xs">
                    {problems.slice(0, 5).map((p) => (
                      <li key={p}>{p}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </div>
        </div>
      </Section>

      <Footer>
        <p aria-live="polite" className="min-w-0 flex-1 truncate text-sm text-fg-secondary">
          {create.isPending ? t.designing : pickSummary(selected.size, freeText, locale)}
        </p>
        <Button variant="primary" onClick={design} disabled={!canDesign} loading={create.isPending}>
          {t.design}
        </Button>
      </Footer>
    </div>
  );
}
