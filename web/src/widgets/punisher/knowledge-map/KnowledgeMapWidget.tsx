import { Suspense, lazy, useCallback, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { Button, EmptyState, ErrorState, ListSkeleton, Sheet, Skeleton, useIsSplit } from "@forgen-team/ui";
import { useConceptDetail, useGoalPath, useGoals, useGraph, useMasteryBatch, useToday } from "@/src/entities/punisher";
import { GoalsManager } from "@/src/features/punisher/goal-manage";
import { KnownAreasSheet } from "@/src/features/punisher/known-onboarding";
import { FOUNDATIONS_AREA, describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { Link, usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { usePageActions, useScreenTitle, type PageAction } from "@/src/shared/lib/screen-chrome";
import { ConceptDetail } from "./ui/ConceptDetail";
import { CHAPTER_OPTION_PREFIX, MapControls, type MapView } from "./ui/MapControls";
import { MapConceptList } from "./ui/MapConceptList";
import { MapLegendSheet } from "./ui/MapPanels";
import { FoundationsToggle } from "./ui/FoundationsToggle";
import { chapterView } from "./lib/chapterView";
import { FOUNDATIONS_SHOWN, hiddenFoundationsCount, visibleFoundations } from "./lib/foundationsView";
import { closeOverlay, openOverlay } from "./lib/overlayHistory";
import { messages } from "./i18n";

const NEW_CHAPTER_PATH = "/chapters/new";
const LazyMapCanvas = lazy(() => import("./ui/MapCanvas"));

function MapCanvas(props: React.ComponentProps<typeof LazyMapCanvas>) {
  return (
    <Suspense fallback={<Skeleton lines={3} className="h-full rounded-none" />}>
      <LazyMapCanvas {...props} />
    </Suspense>
  );
}

/** 기본은 전체 영역(전체가 한 화면에 들어오게 fitView). 영역 필터는 선택 사항. */
const DEFAULT_AREA = "all";

type MapPanel = "goals" | "known";

/**
 * 지식 지도 `/map` (T-canvas): Toolbar 한 줄 + 본문.
 * 폰은 목록 보기(영역별 묶음)가 기본이고 개념을 고르면 전체 화면 상세, 넓은 화면(1024+)은 그래프가 기본이고 오른쪽 옆 패널이 상세다.
 * 주소 = 상태: ?view=list|graph &area= &goal= &node= &panel=goals|known (옛 ?concept= 도 읽는다).
 * `?node=` 는 처음 열 때 push, 열린 채 다른 개념은 replace. `?panel=` 시트는 push 로 열고 닫으면 뒤로 간다.
 * 새 챕터·목표 관리·이미 아는 개념 표시는 셸의 페이지 동작(usePageActions)이다.
 */
export function KnowledgeMapWidget() {
  const t = useT(messages);
  const router = useRouter();
  const pathname = usePathname();
  const { search } = useLocation();
  const params = useSearchParams();
  const split = useIsSplit();
  const goalId = params.get("goal") ?? "";
  const areaParam = params.get("area") ?? DEFAULT_AREA;
  const requestId = params.get("request");
  const chapterParam = params.get("chapter");
  const panelParam = params.get("panel");
  const panel: MapPanel | null = panelParam === "goals" || panelParam === "known" ? panelParam : null;
  const areaFilter = goalId || chapterParam || areaParam === "all" ? null : areaParam;
  // 0단계 기초: 내 수준에서 기본 숨김이면 ?foundations=shown 으로 펼친다(목표·챕터 보기에는 끼지 않는다).
  const foundationsExpanded = params.get("foundations") === FOUNDATIONS_SHOWN;
  const viewParam = params.get("view");
  const view: MapView = viewParam === "graph" || viewParam === "list" ? viewParam : split ? "graph" : "list";

  const graph = useGraph();
  // ?request=<학습 요청 id> (RSS "지도에서 보기") → 그 요청이 붙은 개념을 연다.
  const requestConceptId = useMemo(
    () => (requestId ? graph.data?.nodes.find((n) => n.requests?.some((r) => r.id === requestId))?.id ?? null : null),
    [graph.data, requestId],
  );
  const selectedId = params.get("node") ?? params.get("concept") ?? requestConceptId;
  const selectedNode = useMemo(() => graph.data?.nodes.find((n) => n.id === selectedId), [graph.data, selectedId]);
  const selectedDetail = useConceptDetail(selectedId ?? undefined);
  // 숙달 링: 지도 전체 개념을 한 번에(300개씩) 받아 숙달한 개념에 그린다(영역을 바꿔도 다시 받지 않게 전체 기준).
  const allIds = useMemo(() => (graph.data?.nodes ?? []).map((n) => n.id), [graph.data]);
  const mastery = useMasteryBatch(allIds);
  const masteredIds = useMemo(() => new Set((mastery.data ?? []).filter((m) => m.mastered).map((m) => m.conceptId)), [mastery.data]);
  // 이어보기(최근에 본 순) — 진행 중인 개념이 여럿이면 첫 화면을 가장 최근 것에 맞춘다.
  const today = useToday();
  const recentConceptIds = useMemo(
    () => (today.data?.items ?? []).filter((item) => item.kind === "resume").map((item) => item.conceptId),
    [today.data],
  );
  const goals = useGoals();
  const goalPath = useGoalPath(goalId || undefined);
  const [legendOpen, setLegendOpen] = useState(false);

  const here = `${pathname}${search}`;
  const hrefWith = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(params.toString());
      for (const [key, value] of Object.entries(patch)) {
        if (value) { next.set(key, value); } else { next.delete(key); }
      }
      const qs = next.toString();
      return qs ? `${pathname}?${qs}` : pathname;
    },
    [params, pathname],
  );
  // 보기·영역·목표 같은 필터는 기록에 쌓는다(뒤로 = 앞 필터). 같은 값이면 이동하지 않는다.
  const setParams = useCallback((patch: Record<string, string | null>) => router.push(hrefWith(patch), { scroll: false }), [router, hrefWith]);
  const onSelect = useCallback(
    (id: string) => openOverlay(router, hrefWith({ node: id, concept: null, request: null }), selectedId ? here : undefined),
    [router, hrefWith, selectedId, here],
  );
  const closeDetail = useCallback(
    () => closeOverlay(router, here, hrefWith({ node: null, concept: null, request: null })),
    [router, here, hrefWith],
  );
  const openPanel = useCallback(
    (next: MapPanel) => openOverlay(router, hrefWith({ panel: next, node: null, concept: null, request: null })),
    [router, hrefWith],
  );
  const closePanel = useCallback(() => closeOverlay(router, here, hrefWith({ panel: null })), [router, here, hrefWith]);
  const showGoalPath = useCallback(
    (id: string, conceptId?: string) => setParams({ goal: id, node: conceptId ?? null, concept: null, panel: null }),
    [setParams],
  );

  const fullScreenDetail = !split && Boolean(selectedId);
  useScreenTitle(fullScreenDetail ? (selectedNode?.name ?? selectedDetail.data?.name) : null);
  const pageActions = useMemo<PageAction[] | null>(
    () => fullScreenDetail ? null : [
      { id: "new-chapter", label: t.newChapter, icon: "plus", href: NEW_CHAPTER_PATH, kind: "default" },
      { id: "manage-goals", label: t.manageGoals, onSelect: () => openPanel("goals") },
      { id: "mark-known", label: t.markKnown, onSelect: () => openPanel("known") },
    ],
    [fullScreenDetail, t, openPanel],
  );
  usePageActions(pageActions);

  const pathIds = useMemo(() => {
    if (!goalId || !goalPath.data) { return null; }
    return new Set(goalPath.data.paths.flatMap((p) => (p.steps ?? []).map((s) => s.conceptId)));
  }, [goalId, goalPath.data]);

  const { nodes, edges } = useMemo(() => {
    const all = graph.data?.nodes ?? [];
    let visible = all;
    if (pathIds) { visible = all.filter((n) => pathIds.has(n.id)); }
    else if (chapterParam) { visible = chapterView(all, graph.data?.edges ?? [], chapterParam); }
    else if (areaFilter) { visible = all.filter((n) => n.area === areaFilter); }
    if (!pathIds && !chapterParam) { visible = visibleFoundations(visible, foundationsExpanded, areaFilter); }
    const ids = new Set(visible.map((n) => n.id));
    const prereq = (graph.data?.edges ?? []).filter((edge) => edge.kind === "prerequisite" && ids.has(edge.from) && ids.has(edge.to));
    return { nodes: visible, edges: prereq };
  }, [graph.data, pathIds, areaFilter, chapterParam, foundationsExpanded]);
  const foundationsCount = useMemo(() => hiddenFoundationsCount(graph.data?.nodes ?? []), [graph.data]);
  // 0단계가 내 수준에서 숨김이고 펼치지 않았으면 영역 요약에서도 뺀다(지도와 같은 기준).
  const coverageRows = useMemo(() => {
    const rows = graph.data?.coverage ?? [];
    const hideFoundations = Boolean(graph.data?.foundationsHidden) && !foundationsExpanded;
    return hideFoundations ? rows.filter((c) => c.area !== FOUNDATIONS_AREA) : rows;
  }, [graph.data, foundationsExpanded]);
  const showFoundationsToggle = foundationsCount > 0 && !goalId && !chapterParam && areaFilter === null;
  const mapIsEmpty = graph.isSuccess && (graph.data?.nodes.length ?? 0) === 0;

  function renderCanvas() {
    if (graph.isLoading || (goalId && goalPath.isLoading)) {
      return view === "list" ? <ListSkeleton rows={6} /> : <Skeleton lines={3} className="h-full rounded-none" />;
    }
    if (graph.error || goalPath.error) {
      return (
        <ErrorState
          title={describePunisherError(graph.error ?? goalPath.error)}
          onRetry={() => {
            void graph.refetch();
            if (goalId) { void goalPath.refetch(); }
          }}
        />
      );
    }
    if (mapIsEmpty) {
      return (
        <EmptyState
          title={t.emptyMapTitle}
          action={<Button asChild variant="primary"><Link href={NEW_CHAPTER_PATH}>{t.openChapter}</Link></Button>}
        />
      );
    }
    if (nodes.length === 0) {
      return <EmptyState title={t.emptyTitle} />;
    }
    if (view === "list") {
      return <MapConceptList nodes={nodes} coverage={coverageRows} selectedId={selectedId} onSelect={onSelect} />;
    }
    return (
      <MapCanvas
        nodes={nodes}
        edges={edges}
        highlightIds={pathIds}
        selectedId={selectedId}
        masteredIds={masteredIds}
        recentConceptIds={recentConceptIds}
        onSelect={onSelect}
      />
    );
  }

  const sheets = (
    <>
      <MapLegendSheet open={legendOpen} onClose={() => setLegendOpen(false)} />
      <Sheet open={panel === "goals"} onOpenChange={(o) => { if (!o) { closePanel(); } }} title={t.manageGoals} desktop="dialog">
        <GoalsManager onShowPath={showGoalPath} />
      </Sheet>
      <KnownAreasSheet open={panel === "known"} onClose={closePanel} />
    </>
  );

  // 폰: 개념 상세는 전체 화면(제목 = 개념 이름, ← = 닫기는 셸이 그린다).
  if (fullScreenDetail && selectedId) {
    return (
      <>
        <ConceptDetail conceptId={selectedId} node={selectedNode} variant="screen" onClose={closeDetail} onSelect={onSelect} />
        {sheets}
      </>
    );
  }

  const toolbar = (
    <MapControls
      area={chapterParam ? `${CHAPTER_OPTION_PREFIX}${chapterParam}` : areaParam}
      view={view}
      onViewChange={(v) => setParams({ view: v })}
      chapters={graph.data?.chapters ?? []}
      goalId={goalId}
      goals={goals.data ?? []}
      coverage={graph.data?.coverage ?? []}
      onAreaChange={(a) =>
        a.startsWith(CHAPTER_OPTION_PREFIX)
          ? setParams({ chapter: a.slice(CHAPTER_OPTION_PREFIX.length), area: null })
          : setParams({ area: a, chapter: null })
      }
      onGoalChange={(g) => setParams({ goal: g || null })}
      onOpenLegend={() => setLegendOpen(true)}
    />
  );
  const foundationsToggle = showFoundationsToggle && (
    <FoundationsToggle
      count={foundationsCount}
      expanded={foundationsExpanded}
      onToggle={() => setParams({ foundations: foundationsExpanded ? null : FOUNDATIONS_SHOWN })}
    />
  );

  return (
    // 넓은 화면: 위젯이 화면 남은 높이(Toolbar 포함)를 차지하고, 캔버스 | 상세 패널 줄이 나머지를 채운다.
    <div className="flex w-full flex-col gap-3 lg:h-[calc(100dvh-14rem)] lg:min-h-[28rem]">
      {toolbar}
      {foundationsToggle}
      {split ? (
        <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_20rem] grid-rows-[minmax(0,1fr)] border border-line xl:grid-cols-[minmax(0,1fr)_24rem]">
          <div className={view === "list" ? "min-w-0 overflow-y-auto p-4" : "min-w-0 overflow-hidden bg-canvas-subtle"}>{renderCanvas()}</div>
          <aside aria-label={t.detailLabel} className="min-w-0 overflow-y-auto overflow-x-hidden border-l border-line p-4">
            {selectedId ? (
              <ConceptDetail conceptId={selectedId} node={selectedNode} variant="panel" onClose={closeDetail} onSelect={onSelect} />
            ) : (
              <p className="py-6 text-sm text-fg-muted">{t.detailPlaceholder}</p>
            )}
          </aside>
        </div>
      ) : view === "list" ? (
        <div className="min-w-0">{renderCanvas()}</div>
      ) : (
        <div className="-mx-4 h-[60dvh] min-h-80 overflow-hidden border-y border-line fold:mx-0 fold:border">{renderCanvas()}</div>
      )}
      {sheets}
    </div>
  );
}
