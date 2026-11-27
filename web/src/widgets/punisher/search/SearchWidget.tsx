import { useEffect, useRef, useState } from "react";
import { Search } from "lucide-react";
import { useRouter, useSearchParams } from "@/src/shared/lib/router";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton, TextField } from "@forgen-team/ui";
import { cn } from "@/src/shared/lib/cn";
import { standaloneColumnClass } from "@/src/shared/config/runtime";
import { useSearch } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { groupHits, hitCounts, hitHref, snippetParts } from "./lib/hits";
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
  // ui TextField 의 inputRef 는 input·textarea 겸용 타입이다.
  const inputRef = useRef<HTMLInputElement & HTMLTextAreaElement>(null);

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
    body = <ErrorState title={describePunisherError(result.error)} onRetry={() => void result.refetch()} />;
  } else if (q && result.data && result.data.length === 0) {
    body = <EmptyState title={t.empty(q)} />;
  } else if (q && result.data) {
    const groups = groupHits(result.data);
    const counts = hitCounts(groups);
    const snippet = (text: string) =>
      snippetParts(text, q).map((part, i) =>
        part.match ? <mark key={i} className="rounded-sm bg-accent-subtle px-0.5 text-accent-text">{part.text}</mark> : <span key={i}>{part.text}</span>);
    body = (
      <>
        <p aria-live="polite" className="text-xs text-fg-muted">{t.count(counts.lectures, counts.places, counts.others)}</p>
        <div className="flex flex-col gap-4">
          {groups.map((g) =>
            g.type === "lecture" ? (
              // 같은 강의의 결과는 한 묶음: 강의 제목 1줄 + 섹션 이름·발췌(각각 그 섹션 ?at= 링크).
              <section key={`lecture:${g.lectureId}`} aria-label={t.lectureGroup(g.title, g.hits.length)} className="flex flex-col">
                <h2 className="flex items-baseline gap-2 px-1 pb-1 text-sm font-semibold text-fg">
                  <span className="shrink-0 text-xs font-medium text-fg-secondary">{t.kind("lecture")}</span>
                  <span className="min-w-0 truncate">{g.title}</span>
                </h2>
                <List>
                  {g.hits.map((hit) => (
                    <ListRow
                      key={`${hit.refId}:${hit.anchor}`}
                      href={hitHref(hit)}
                      title={hit.sectionTitle || t.untitledSection}
                      metaLines={2}
                      meta={snippet(hit.snippet)}
                    />
                  ))}
                </List>
              </section>
            ) : (
              <List key={`${g.hit.kind}:${g.hit.refId}`}>
                <ListRow
                  href={hitHref(g.hit)}
                  title={g.hit.title || t.kind(g.hit.kind)}
                  metaLines={2}
                  meta={
                    <>
                      <span className="mr-1 font-medium text-fg-secondary">{t.kind(g.hit.kind)}</span>
                      {snippet(g.hit.snippet)}
                    </>
                  }
                />
              </List>
            ),
          )}
        </div>
      </>
    );
  }

  return (
    <div className={cn("flex w-full flex-col gap-3", standaloneColumnClass())}>
      <div role="search">
        <TextField
          inputRef={inputRef}
          prefixIcon={<Search aria-hidden="true" />}
          placeholder={t.placeholder}
          value={text}
          onValueChange={({ value }) => change(value)}
          hideCharacterCount
          inputProps={{ type: "search", "aria-label": t.label, maxLength: 100 }}
        />
      </div>
      {!q && <p className="text-xs text-fg-muted">{t.hint}</p>}
      {body}
    </div>
  );
}
