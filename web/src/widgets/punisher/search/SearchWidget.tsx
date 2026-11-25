import { useEffect, useRef, useState } from "react";
import { Search } from "lucide-react";
import { Link, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { EmptyState, ErrorState, Input, List, ListRow, ListSkeleton } from "@/src/shared/ui/kit";
import { useSearch } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { hitHref, snippetParts } from "./lib/hits";
import { messages } from "./i18n";

const DEBOUNCE_MS = 300;

/**
 * 학습 자료 전문 검색(W5a): 강의(섹션)·메모·강의 중 질문·질문 해설. 검색어는 주소(?q=)에 남는다(화면=주소).
 * 결과는 그 자리로 가는 링크(강의는 ?at= 섹션).
 */
export function SearchWidget() {
  const t = useT(messages);
  const router = useRouter();
  const params = useSearchParams();
  const q = params.get("q") ?? "";
  const [text, setText] = useState(q);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const result = useSearch(q);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => { inputRef.current?.focus(); }, []);
  useEffect(() => {
    setText((prev) => (prev.trim() === q ? prev : q));
  }, [q]);
  useEffect(() => () => { if (timer.current) { clearTimeout(timer.current); } }, []);

  function change(value: string) {
    setText(value);
    if (timer.current) { clearTimeout(timer.current); }
    timer.current = setTimeout(() => {
      const v = value.trim();
      router.push(v ? `/search?q=${encodeURIComponent(v)}` : "/search", { scroll: false });
    }, DEBOUNCE_MS);
  }

  let body = null;
  if (q && result.isLoading) {
    body = <ListSkeleton rows={4} />;
  } else if (q && result.error) {
    body = <ErrorState message={describePunisherError(result.error)} onRetry={() => result.refetch()} retrying={result.isRefetching} />;
  } else if (q && result.data && result.data.length === 0) {
    body = <EmptyState compact title={t.empty(q)} />;
  } else if (q && result.data) {
    body = (
      <>
        <p aria-live="polite" className="text-xs text-fg-muted">{t.count(result.data.length)}</p>
        <List>
          {result.data.map((hit) => (
            <ListRow
              key={`${hit.kind}:${hit.refId}:${hit.anchor}`}
              href={hitHref(hit)}
              linkAs={Link}
              title={hit.title || t.kind(hit.kind)}
              titleLines={1}
              metaLines="all"
              meta={
                <>
                  <span className="mr-1 font-medium text-fg-secondary">{t.kind(hit.kind)}</span>
                  {snippetParts(hit.snippet, q).map((part, i) =>
                    part.match ? <mark key={i} className="rounded-sm bg-accent-subtle px-0.5 text-accent-text">{part.text}</mark> : <span key={i}>{part.text}</span>)}
                </>
              }
            />
          ))}
        </List>
      </>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-3">
      <div role="search" className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-fg-muted" aria-hidden="true" />
        <Input ref={inputRef} type="search" aria-label={t.label} placeholder={t.placeholder} value={text} maxLength={100}
          onChange={(e) => change(e.target.value)} className="pl-9" />
      </div>
      {!q && <p className="text-xs text-fg-muted">{t.hint}</p>}
      {body}
    </div>
  );
}
