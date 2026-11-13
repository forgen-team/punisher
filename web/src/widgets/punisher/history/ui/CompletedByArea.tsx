import { Link } from "@/src/shared/lib/router";
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { useEffect, useRef, useState } from "react";
import { ChevronRight, Search } from "lucide-react";
import { Chip, EmptyState, ErrorState, Input, List, ListRow, ListSkeleton, cn, focusRing } from "@/src/shared/ui/kit";
import { InquiryStatusIcon, areaLabel, groupByParent, useCompleted, useCompletedAreas } from "@/src/entities/punisher";
import { inquiryHref, lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type CompletedArea, type CompletedItem } from "@/src/shared/api/punisher";
import { DeeperGroup } from "@/src/shared/ui/punisher";
import { LoadMore } from "./LoadMore";
import { historyDay } from "../lib/historyDate";

const DEBOUNCE_MS = 300;
/** 전체 보기에서 처음부터 펼쳐 두는 분야 수(접힌 분야는 펼칠 때 불러온다). */
const OPEN_BY_DEFAULT = 2;

function CompletedRow({ item }: { item: CompletedItem }) {
  const date = historyDay(item.lastActivityAt);
  if (item.kind === "inquiry") {
    return (
      <ListRow
        href={inquiryHref(item.id)}
        linkAs={Link}
        leading={<InquiryStatusIcon status="answered" />}
        title={item.title}
        meta={["질문 해설", item.followupCount ? `이어서 ${item.followupCount}번` : "", date].filter(Boolean).join(" · ")}
      />
    );
  }
  return (
    <ListRow
      href={lectureHref(item.id)}
      linkAs={Link}
      title={item.title || item.conceptName}
      meta={[item.conceptName && item.conceptName !== item.title ? item.conceptName : "", item.lectureKind === "instant" ? "즉석" : "라이브러리", date].filter(Boolean).join(" · ")}
    />
  );
}

/** 분야 한 칸의 목록 — 펼쳤을 때만 서버에서 불러온다. 원 강의 + '더 깊게' 묶음은 한 덩어리로 둔다. */
function AreaItems({ area, q, twoColumns }: { area: string; q: string; twoColumns: boolean }) {
  const completed = useCompleted({ area, q });
  const items = completed.data?.pages.flatMap((p) => p.items) ?? [];
  if (completed.isLoading) { return <ListSkeleton rows={3} />; }
  if (completed.error) {
    return <ErrorState compact message={describePunisherError(completed.error)} onRetry={() => completed.refetch()} />;
  }
  const groups = groupByParent(items, (it) => it.id, (it) => it.parentLectureId);
  return (
    <div className="flex flex-col gap-3">
      <div className={cn(twoColumns && "xl:grid xl:grid-cols-2 xl:gap-x-8 xl:[&>ul:nth-last-child(2):nth-child(odd)]:border-b-0")}>
        {groups.map(({ item, children }) => (
          <List key={item.id} className="border-b border-line last:border-b-0">
            <DeeperGroup parent={<CompletedRow item={item} />}>
              {children.map((c) => (
                <CompletedRow key={c.id} item={c} />
              ))}
            </DeeperGroup>
          </List>
        ))}
      </div>
      <LoadMore hasNextPage={Boolean(completed.hasNextPage)} isFetching={completed.isFetchingNextPage} onLoad={() => completed.fetchNextPage()} />
    </div>
  );
}

function AreaSection({ area, q, defaultOpen, single }: { area: CompletedArea; q: string; defaultOpen: boolean; single: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  const expanded = single || open;
  return (
    <section className={cn("min-w-0 rounded-xl border border-line bg-canvas-elevated px-4 shadow-xs", single && "xl:col-span-2")}>
      <h2>
        <button
          type="button"
          aria-expanded={expanded}
          disabled={single}
          onClick={() => setOpen((value) => !value)}
          className={cn("flex min-h-touch w-full items-center gap-2 rounded-md text-left", focusRing)}
        >
          {!single && <ChevronRight className={cn("size-4 shrink-0 text-fg-muted transition-transform", expanded && "rotate-90")} aria-hidden="true" />}
          <span className="min-w-0 flex-1 break-keep text-sm font-semibold text-fg">{areaLabel(area.area)}</span>
          <span className="shrink-0 text-xs tabular-nums text-fg-muted">{area.count}개</span>
        </button>
      </h2>
      {expanded && (
        <div className="border-t border-line py-2">
          <AreaItems area={area.area} q={q} twoColumns={single} />
        </div>
      )}
    </section>
  );
}

/**
 * 완료한 강의·질문 해설을 지도의 분야별로 — 검색(제목·개념명)과 분야 칩(하나만)은 서버 쿼리, 주소에 남는다(?q=&area=).
 * 전체 보기는 분야별 접기/펼치기, 넓은 화면(xl)은 분야 두 열, 분야 하나를 고르면 전체 폭에 목록 두 열.
 */
export function CompletedByArea() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const urlQuery = params.get("q") ?? "";
  const selected = params.get("area") ?? "";
  const [text, setText] = useState(urlQuery);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const areas = useCompletedAreas(urlQuery);

  function setParam(patch: Record<string, string>) {
    const next = new URLSearchParams(window.location.search);
    for (const [key, value] of Object.entries(patch)) {
      if (value) { next.set(key, value); } else { next.delete(key); }
    }
    router.replace(`${pathname}?${next.toString()}`, { scroll: false });
  }

  function changeText(value: string) {
    setText(value);
    if (timer.current) { clearTimeout(timer.current); }
    timer.current = setTimeout(() => setParam({ q: value.trim() }), DEBOUNCE_MS);
  }

  useEffect(() => () => {
    if (timer.current) { clearTimeout(timer.current); }
  }, []);

  const list = areas.data ?? [];
  const total = list.reduce((sum, a) => sum + a.count, 0);
  const shown = selected ? list.filter((a) => a.area === selected) : list;

  let body;
  if (areas.isLoading) {
    body = <ListSkeleton rows={5} />;
  } else if (areas.error) {
    body = <ErrorState message={describePunisherError(areas.error)} onRetry={() => areas.refetch()} retrying={areas.isRefetching} />;
  } else if (shown.length === 0) {
    body = urlQuery || selected ? (
      <EmptyState title="찾는 기록이 없어요" description={selected ? "다른 말로 찾거나 '전체'로 바꿔 보세요." : "다른 말로 찾아 보세요."} />
    ) : (
      <EmptyState title="아직 완료한 기록이 없어요" description="강의를 끝까지 읽거나 질문 해설을 받으면 여기에 쌓여요." />
    );
  } else {
    body = (
      <div className="grid gap-4 xl:grid-cols-2 xl:items-start xl:gap-6">
        {shown.map((a, index) => (
          <AreaSection key={`${a.area}:${urlQuery}`} area={a} q={urlQuery} defaultOpen={index < OPEN_BY_DEFAULT} single={Boolean(selected)} />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div role="search" className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-fg-muted" aria-hidden="true" />
        <Input aria-label="완료 기록 찾기" value={text} onChange={(event) => changeText(event.target.value)} placeholder="제목이나 개념 이름으로 찾기" className="pl-9" />
      </div>
      {/* 칩 줄은 불러오는 중에도, 결과가 없어도 자리를 지켜 아래가 밀리지 않는다. */}
      <div role="group" aria-label="분야" aria-busy={areas.isLoading} className="-mx-4 flex min-h-11 gap-2 overflow-x-auto px-4 py-1.5 fold:mx-0 fold:flex-wrap fold:px-0">
        <Chip selected={!selected} onClick={() => setParam({ area: "" })}>
          전체{areas.isLoading ? "" : ` ${total}`}
        </Chip>
        {list.map((a) => (
          <Chip key={a.area} selected={selected === a.area} onClick={() => setParam({ area: selected === a.area ? "" : a.area })}>
            {areaLabel(a.area)} {a.count}
          </Chip>
        ))}
      </div>
      {body}
    </div>
  );
}
