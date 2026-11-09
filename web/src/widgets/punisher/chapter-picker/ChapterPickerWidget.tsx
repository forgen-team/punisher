
import { useRouter } from "@/src/shared/lib/router";
import { useMemo, useState } from "react";
import { Sparkles } from "lucide-react";
import { Button, ErrorState, Field, ListSkeleton, Section, Textarea } from "@/src/shared/ui/kit";
import { useApplyChapterDraft, useChapterSuggestions, useCreateChapterDraft, useGraph } from "@/src/entities/punisher";
import { PunisherApiError, describePunisherError } from "@/src/shared/api/punisher";
import { DraftPreview } from "./ui/DraftPreview";
import { SuggestionCards } from "./ui/SuggestionCards";
import { FREE_TEXT_MAX, MAX_PICKS } from "./ui/constants";

/** 422 CHAPTER_INVALID 이면 서버가 준 문제 목록을 함께 보여 준다. */
function designProblems(error: unknown): string[] {
  return error instanceof PunisherApiError && error.status === 422 ? error.problems : [];
}

/**
 * 다음 챕터 고르기(질문형): 자동 제안 카드(여러 개) + 직접 입력 → 설계(미리보기) → '이대로 열기'.
 * 생성은 Claude 게이트웨이로 하고, 반영 전까지 지도는 바뀌지 않는다.
 */
export function ChapterPickerWidget() {
  const router = useRouter();
  const suggestions = useChapterSuggestions();
  const graph = useGraph();
  const create = useCreateChapterDraft();
  const apply = useApplyChapterDraft();
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [freeText, setFreeText] = useState("");

  const names = useMemo(() => new Map((graph.data?.nodes ?? []).map((n) => [n.id, n.name])), [graph.data]);

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) { next.delete(id); } else if (next.size < MAX_PICKS) { next.add(id); }
      return next;
    });
  }

  function design() {
    create.mutate({ suggestionIds: [...selected], freeText: freeText.trim() });
  }

  function openChapter() {
    if (!create.data) { return; }
    apply.mutate(create.data.draftId, {
      onSuccess: (res) => router.push(`/map?chapter=${encodeURIComponent(res.chapterId)}`),
    });
  }

  if (create.data) {
    return (
      <div className="flex flex-col gap-3">
        <DraftPreview
          draft={create.data.draft}
          names={names}
          applying={apply.isPending}
          onApply={openChapter}
          onBack={() => {
            create.reset();
            apply.reset();
          }}
        />
        {apply.error && (
          <p role="alert" className="text-sm text-danger">
            {describePunisherError(apply.error)}
          </p>
        )}
      </div>
    );
  }

  const canDesign = (selected.size > 0 || freeText.trim() !== "") && !create.isPending;
  const problems = designProblems(create.error);

  return (
    <div className="flex flex-col gap-5">
      <header className="flex flex-col gap-1 pt-2">
        <h1 className="text-xl font-semibold text-fg">{suggestions.data?.question ?? "다음엔 어떤 걸 공부해 보고 싶어요?"}</h1>
        <p className="text-sm text-fg-secondary">
          제안 중에서 여러 개를 골라도 되고, 직접 적어도 돼요. 고른 것으로 개념 5~12개짜리 챕터를 설계해 미리 보여 드려요.
        </p>
      </header>

      <Section title="추천" description={`최대 ${MAX_PICKS}개까지 고를 수 있어요`}>
        {suggestions.isLoading ? (
          <ListSkeleton rows={4} />
        ) : suggestions.error ? (
          <ErrorState compact message={describePunisherError(suggestions.error)} onRetry={() => suggestions.refetch()} />
        ) : (
          <SuggestionCards suggestions={suggestions.data?.suggestions ?? []} selected={selected} onToggle={toggle} />
        )}
      </Section>

      <Field label="직접 입력" hint={`${freeText.length}/${FREE_TEXT_MAX}자`}>
        <Textarea
          value={freeText}
          maxLength={FREE_TEXT_MAX}
          rows={3}
          placeholder="예) Kubernetes 네트워킹을 실무 수준으로"
          onChange={(event) => setFreeText(event.target.value)}
        />
      </Field>

      <div aria-live="polite" className="empty:hidden">
        {create.error && (
          <div role="alert" className="rounded-md bg-danger-subtle px-3 py-2 text-sm text-danger">
            <p>{describePunisherError(create.error)}</p>
            {problems.length > 0 && (
              <ul className="mt-1 list-disc pl-5 text-xs">
                {problems.slice(0, 5).map((p) => (
                  <li key={p}>{p}</li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      <div className="flex justify-end">
        <Button onClick={design} disabled={!canDesign} loading={create.isPending}>
          <Sparkles className="size-4" aria-hidden="true" />
          {create.isPending ? "챕터를 설계하는 중… (1~3분)" : "챕터 설계해 보기"}
        </Button>
      </div>
    </div>
  );
}
