
import { useState, type FormEvent } from "react";
import { ArrowRight, Search } from "lucide-react";
import { IconButton, Input, Spinner } from "@/src/shared/ui/kit";
import { InlineError } from "@/src/shared/ui/punisher";
import type { InstantResponse } from "@/src/shared/api/punisher";
import { ResolutionChoices } from "./ResolutionChoices";
import { useLaunchInstant } from "./useLaunchInstant";

const MAX_QUERY = 500;

/** "궁금한 것 입력" 한 줄 — 자유 질의로 즉석 강의 시작 (design §3.2 B). */
export function InstantQueryBox() {
  const [query, setQuery] = useState("");
  const [submitted, setSubmitted] = useState("");
  const [result, setResult] = useState<InstantResponse | null>(null);
  const { launch, isPending, error } = useLaunchInstant();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const trimmed = query.trim();
    if (!trimmed || isPending) { return; }
    setResult(null);
    setSubmitted(trimmed);
    try {
      setResult(await launch({ query: trimmed }));
    } catch {
      // 오류는 아래 <InlineError error={error}> 가 표시한다.
      setResult(null);
    }
  }

  async function handlePick(conceptId: string) {
    try {
      setResult(await launch({ conceptId, query: submitted }));
    } catch {
      // 선택지를 유지하고 오류만 표시한다.
      return;
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <form role="search" onSubmit={handleSubmit} className="relative flex items-center">
        <Search className="pointer-events-none absolute left-3 size-4 text-fg-muted" aria-hidden="true" />
        <Input
          aria-label="궁금한 것 입력"
          value={query}
          maxLength={MAX_QUERY}
          enterKeyHint="search"
          onChange={(event) => setQuery(event.target.value)}
          placeholder="궁금한 것 입력 (예: Pod가 죽었다)"
          className="pl-9 pr-12"
        />
        <IconButton
          type="submit"
          size="sm"
          aria-label={isPending ? "해석 중" : "강의 찾기"}
          disabled={isPending || !query.trim()}
          className="absolute right-1.5"
        >
          {isPending ? <Spinner size="sm" label={null} /> : <ArrowRight aria-hidden="true" />}
        </IconButton>
      </form>
      <div aria-live="polite" className="flex flex-col gap-2">
        <InlineError error={error} />
        {result && <ResolutionChoices response={result} disabled={isPending} onPick={handlePick} />}
      </div>
    </div>
  );
}
