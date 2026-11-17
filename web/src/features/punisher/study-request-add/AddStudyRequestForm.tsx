import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";
import { Button, Input } from "@/src/shared/ui/kit";
import { useCreateStudyRequest } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

const MAX_QUERY = 500;

/**
 * 단독판 전용: 학습 요청 직접 추가(witems 에서는 RSS '30분 학습'이 이 역할).
 * 추가하면 서버가 바로 개념을 찾고, 찾으면 그 주제 강의를 먼저 미리 만든다.
 */
export function AddStudyRequestForm() {
  const t = useT(messages);
  const [query, setQuery] = useState("");
  const create = useCreateStudyRequest();
  const trimmed = query.trim();

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!trimmed || create.isPending) { return; }
    create.mutate(trimmed, { onSuccess: () => setQuery("") });
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-1 pb-1 pt-4">
      <div className="flex items-center gap-2">
        <Input
          aria-label={t.topicLabel}
          className="min-w-0"
          placeholder={t.placeholder}
          value={query}
          maxLength={MAX_QUERY}
          onChange={(event) => setQuery(event.target.value)}
        />
        <Button type="submit" size="md" variant="secondary" className="shrink-0" loading={create.isPending} disabled={!trimmed}>
          <Plus className="h-4 w-4" aria-hidden="true" />
          {t.add}
        </Button>
      </div>
      <InlineError error={create.error} />
    </form>
  );
}
