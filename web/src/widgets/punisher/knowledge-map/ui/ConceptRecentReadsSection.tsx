import { useQuery } from "@tanstack/react-query";
import { Newspaper } from "lucide-react";
import { List, ListRow, Section } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface RecentReads {
  count: number;
  items: Array<{ itemId: string; title: string; feedTitle: string }>;
}

async function fetchRecentReads(conceptId: string): Promise<RecentReads> {
  const res = await fetch(`/api/rss/v1/reading/concepts/${encodeURIComponent(conceptId)}/recent`, {
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (!res.ok) {
    throw new Error(`recent reads ${res.status}`);
  }
  return (await res.json()) as RecentReads;
}

/**
 * 이 개념이 나온, 최근에 연 RSS 글. 분류된 글이 있는 개념에만 보이고 RSS 쪽 실패·로딩 중에는 아무것도 그리지 않는다
 * (시트 맨 아래라 나타나도 위 내용이 밀리지 않는다).
 */
export function ConceptRecentReadsSection({ conceptId }: { conceptId: string }) {
  const t = useT(messages);
  const { data } = useQuery({
    queryKey: ["punisher-map", "rss-concept-recent", conceptId],
    queryFn: () => fetchRecentReads(conceptId),
    staleTime: 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
  if (!data || data.count === 0) { return null; }
  return (
    <Section title={t.recentReads(data.count)} headingLevel={3}>
      <List>
        {data.items.map((it) => (
          <ListRow
            key={it.itemId}
            href={`/workspace/rss/feed?item=${encodeURIComponent(it.itemId)}`}
            leading={<Newspaper aria-hidden="true" />}
            title={it.title || t.untitled}
            meta={it.feedTitle}
          />
        ))}
      </List>
    </Section>
  );
}
