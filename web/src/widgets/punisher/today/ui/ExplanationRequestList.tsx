import { useQuery } from "@tanstack/react-query";
import { BookOpenText } from "lucide-react";
import { EmptyState, ErrorState, List, ListRow, ListSkeleton } from "@forgen-team/ui";
import { translate, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** rss-server models.ItemExplanation 중 목록에 쓰는 필드 */
interface ExplanationSummary {
  id: string;
  itemId: string;
  title: string;
  feedTitle: string;
  status: "pending" | "succeeded" | "failed";
  content?: { summary: string } | null;
}

const EXPLAIN_LIMIT = 20;

async function fetchExplanations(): Promise<ExplanationSummary[]> {
  const res = await fetch(`/api/rss/v1/explanations?limit=${EXPLAIN_LIMIT}`, { credentials: "include", headers: { Accept: "application/json" } });
  if (!res.ok) {
    throw new Error(translate(messages).explanationsLoadFailedStatus(res.status));
  }
  return ((await res.json()) as { explanations: ExplanationSummary[] | null }).explanations ?? [];
}

/**
 * 글 해설 — RSS '이 글 깊게 읽기'로 만든 해설을 다시 연다(저장된 해설, 재생성 없음).
 * 예전 '학습 요청'(개념 강의 대기열) 자리를 대신한다. 개념 강의는 해설 끝 '이어서 배울 개념'에서 바로 연다.
 */
export function ExplanationRequestList() {
  const t = useT(messages);
  const { data, isLoading, error, refetch, isRefetching } = useQuery({
    queryKey: ["punisher-today", "rss-explanations"],
    queryFn: fetchExplanations,
    staleTime: 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
  });
  if (isLoading) {
    return <ListSkeleton rows={2} />;
  }
  if (error) {
    return <ErrorState title={t.explanationsLoadFailed} onRetry={() => refetch()} />;
  }
  if (!data || data.length === 0) {
    return <EmptyState title={t.noExplanations} description={t.noExplanationsDescription} />;
  }
  return (
    <List>
        {data.map((explanation) => (
          <ListRow
            key={explanation.id}
            href={`/workspace/rss/explain/${encodeURIComponent(explanation.itemId)}`}
            leading={<BookOpenText className="h-4 w-4 text-fg-muted" aria-hidden />}
            title={explanation.title}
            meta={[explanation.status === "pending" ? t.generating : explanation.content?.summary ?? "", explanation.feedTitle].filter(Boolean).join(" · ")}
          />
        ))}
    </List>
  );
}
