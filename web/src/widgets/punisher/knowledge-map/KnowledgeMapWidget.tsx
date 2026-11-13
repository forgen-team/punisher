
import { usePathname, useRouter, useSearchParams } from "@/src/shared/lib/router";
import { Suspense, lazy, useCallback, useMemo, useState } from "react";
import { BottomSheet, EmptyState, ErrorState, Skeleton } from "@/src/shared/ui/kit";
import { useFrontier, useGoalPath, useGoals, useGraph, useMastery } from "@/src/entities/punisher";
import { GoalsManager } from "@/src/features/punisher/goal-manage";
import { KnownAreasSheet, KnownOnboardingPrompt, useKnownOnboardingPrompt } from "@/src/features/punisher/known-onboarding";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useWideLayout } from "@/src/shared/lib/useWideLayout";
import { ConceptDetailSheet } from "./ui/ConceptDetailSheet";
import { CHAPTER_OPTION_PREFIX, MapControls } from "./ui/MapControls";
import { chapterView } from "./lib/chapterView";

const NEW_CHAPTER_PATH = "/chapters/new";
import { CoverageList, FrontierList, MapLegendSheet, PendingRequestList } from "./ui/MapPanels";

const LazyMapCanvas = lazy(() => import("./ui/MapCanvas"));

function MapCanvas(props: React.ComponentProps<typeof LazyMapCanvas>) {
  return (
    <Suspense fallback={<Skeleton variant="block" className="h-full rounded-none" />}>
      <LazyMapCanvas {...props} />
    </Suspense>
  );
}

/** 기본은 전체 영역(전체가 한 화면에 들어오게 fitView). 영역 필터는 선택 사항. */
const DEFAULT_AREA = "all";

/**
 * 지식 지도: 모바일은 컨트롤 한 줄 + 전체 폭 캔버스, 그 아래 "다음에 공부하면 좋은 것"·커버리지.
 * 노드 상세·범례·목표 관리·아는 개념 표시는 시트. URL 상태(?area=&goal=&concept=&request=&panel=goals).
 * 그래프는 항상 전체를 한 번 받아(약 170개) 영역은 클라이언트에서 거른다 — 영역 요약(학습·요청 수)도 함께 쓴다.
 */
export function KnowledgeMapWidget() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const goalId = params.get("goal") ?? "";
  const areaParam = params.get("area") ?? DEFAULT_AREA;
  const requestId = params.get("request");
  const goalsOpen = params.get("panel") === "goals";
  const chapterParam = params.get("chapter");
  const areaFilter = goalId || chapterParam || areaParam === "all" ? null : areaParam;

  const graph = useGraph();
  // ?request=<학습 요청 id> (RSS "지도에서 보기") → 그 요청이 붙은 개념을 연다.
  const requestConceptId = useMemo(
    () => (requestId ? graph.data?.nodes.find((n) => n.requests?.some((r) => r.id === requestId))?.id ?? null : null),
    [graph.data, requestId],
  );
  const selectedId = params.get("concept") ?? requestConceptId;
  // 지도 전체의 숙달을 한 번에 주는 API 가 없어, 숙달 링은 상세를 연 개념에만 그린다.
  const mastery = useMastery(selectedId ?? undefined);
  const masteredId = mastery.data?.mastered ? mastery.data.conceptId : null;
  const frontier = useFrontier(10);
  const goals = useGoals();
  const goalPath = useGoalPath(goalId || undefined);
  const onboarding = useKnownOnboardingPrompt();
  const [legendOpen, setLegendOpen] = useState(false);
  const [knownOpen, setKnownOpen] = useState(false);

  const setParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(params.toString());
      for (const [key, value] of Object.entries(patch)) {
        if (value) { next.set(key, value); } else { next.delete(key); }
      }
      const qs = next.toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [params, pathname, router],
  );
  const onSelect = useCallback((id: string) => setParams({ concept: id }), [setParams]);
  const closeDetail = useCallback(() => setParams({ concept: null, request: null }), [setParams]);
  const wide = useWideLayout();
  const showGoalPath = useCallback(
    (id: string, conceptId?: string) => setParams({ goal: id, concept: conceptId ?? null, panel: null }),
    [setParams],
  );

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
    const ids = new Set(visible.map((n) => n.id));
    const prereq = (graph.data?.edges ?? []).filter((edge) => edge.kind === "prerequisite" && ids.has(edge.from) && ids.has(edge.to));
    return { nodes: visible, edges: prereq };
  }, [graph.data, pathIds, areaFilter, chapterParam]);
  const selectedRequests = useMemo(
    () => graph.data?.nodes.find((n) => n.id === selectedId)?.requests ?? [],
    [graph.data, selectedId],
  );

  function renderCanvas() {
    if (graph.isLoading || (goalId && goalPath.isLoading)) {
      return <Skeleton variant="block" className="h-full rounded-none" />;
    }
    if (graph.error || goalPath.error) {
      return (
        <ErrorState
          compact
          message={describePunisherError(graph.error ?? goalPath.error)}
          onRetry={() => {
            void graph.refetch();
            if (goalId) { void goalPath.refetch(); }
          }}
        />
      );
    }
    if (nodes.length === 0) {
      return <EmptyState compact title="표시할 개념이 없습니다" description="다른 영역이나 목표를 선택해 보세요." />;
    }
    return <MapCanvas nodes={nodes} edges={edges} highlightIds={pathIds} selectedId={selectedId} masteredId={masteredId} onSelect={onSelect} />;
  }

  return (
    <div className="flex w-full flex-col gap-3">
      <MapControls
        area={chapterParam ? `${CHAPTER_OPTION_PREFIX}${chapterParam}` : areaParam}
        chapters={graph.data?.chapters ?? []}
        goalId={goalId}
        goals={goals.data ?? []}
        coverage={graph.data?.coverage ?? []}
        onAreaChange={(a) =>
          a.startsWith(CHAPTER_OPTION_PREFIX)
            ? setParams({ chapter: a.slice(CHAPTER_OPTION_PREFIX.length), area: null })
            : setParams({ area: a, chapter: null })
        }
        onOpenChapter={() => router.push(NEW_CHAPTER_PATH)}
        onGoalChange={(g) => setParams({ goal: g || null })}
        onOpenLegend={() => setLegendOpen(true)}
        onOpenGoals={() => setParams({ panel: "goals" })}
        onOpenKnown={() => setKnownOpen(true)}
      />
      {onboarding.show && <KnownOnboardingPrompt onStart={() => setKnownOpen(true)} onDismiss={onboarding.dismiss} />}
      {goalId && <p className="text-xs text-fg-muted">목표에 필요한 선행 사슬만 표시합니다.</p>}

      {/* lg: 캔버스 | 18rem 목록, 상세는 오른쪽 시트. xl: 캔버스 | 22rem(2xl 26rem) 열 — 개념을 고르면 그 열이 상세 패널로 바뀐다(캔버스 폭 그대로). */}
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_18rem] xl:grid-cols-[minmax(0,1fr)_22rem] 2xl:grid-cols-[minmax(0,1fr)_26rem]">
        <div className="-mx-4 h-[60dvh] min-h-80 overflow-hidden border-y border-line fold:mx-0 fold:rounded-lg fold:border lg:h-[calc(100dvh-14rem)]">
          {renderCanvas()}
        </div>
        <div className="flex flex-col gap-6 xl:sticky xl:top-[calc(var(--app-header-offset)+1rem)] xl:max-h-[calc(100dvh-var(--app-header-offset)-2rem)] xl:self-start xl:overflow-y-auto">
          {wide && selectedId ? (
            <ConceptDetailSheet variant="panel" conceptId={selectedId} requests={selectedRequests} onClose={closeDetail} onSelect={onSelect} />
          ) : (
            <>
              <PendingRequestList requests={graph.data?.pendingRequests ?? []} highlightId={requestId} />
              <FrontierList items={frontier.data?.items ?? []} onSelect={onSelect} />
              <CoverageList coverage={graph.data?.coverage ?? []} />
            </>
          )}
        </div>
      </div>

      <ConceptDetailSheet
        conceptId={wide === false ? selectedId : null}
        requests={selectedRequests}
        onClose={closeDetail}
        onSelect={onSelect}
      />
      <MapLegendSheet open={legendOpen} onClose={() => setLegendOpen(false)} />
      <BottomSheet open={goalsOpen} onClose={() => setParams({ panel: null })} title="목표 관리" desktop="side">
        <GoalsManager onShowPath={showGoalPath} />
      </BottomSheet>
      <KnownAreasSheet open={knownOpen} onClose={() => setKnownOpen(false)} onSaved={onboarding.dismiss} />
    </div>
  );
}
