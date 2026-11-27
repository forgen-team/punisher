import { useCallback, useEffect, useState } from "react";
import { useBatchMarkKnown, useConcepts, useLearningSettings, useSaveLearningSettings, type MarkKnownProgress } from "@/src/entities/punisher";
import {
  DEFAULT_READER_LEVEL,
  FAMILIARITY_AREAS,
  type ChosenReaderLevel,
  type Concept,
  type Familiarity,
  type FamiliarityArea,
} from "@/src/shared/api/punisher";

const CLOSED_KEY = "punisher.firstSetup.closed";

export type FamiliarityByArea = Record<FamiliarityArea, Familiarity>;

/** 수준에서 고른 영역별 기본값 — 처음 설정 2단계의 첫 선택. */
export function familiarityForLevel(level: ChosenReaderLevel): Familiarity {
  if (level === "advanced") { return "can_explain"; }
  if (level === "practitioner") { return "heard"; }
  return "new";
}

export function defaultFamiliarity(level: ChosenReaderLevel): FamiliarityByArea {
  const value = familiarityForLevel(level);
  return Object.fromEntries(FAMILIARITY_AREAS.map((area) => [area, value])) as FamiliarityByArea;
}

export function explainableAreas(familiarity: FamiliarityByArea): FamiliarityArea[] {
  return FAMILIARITY_AREAS.filter((area) => familiarity[area] === "can_explain");
}

/** 0단계 기초 트랙을 기본 경로 시작점으로 둘 수준(설계 ②). 지금은 요약에 안내만 한다. */
export function startsWithFoundations(level: ChosenReaderLevel): boolean {
  return level === "beginner" || level === "novice";
}

/** '설명 가능' 영역의 깊이 1 활성 개념 중 아직 완료·앎이 아닌 것. */
export function depthOneIdsForAreas(concepts: Concept[], areas: readonly string[]): string[] {
  const wanted = new Set(areas);
  return concepts
    .filter((c) => c.depth === 1 && c.status === "active" && wanted.has(c.area))
    .filter((c) => c.userState !== "completed" && c.userState !== "known")
    .map((c) => c.id);
}

/** 건너뛰거나 × 로 닫으면 이 기기에서 2주 동안 다시 띄우지 않고, 2주 뒤 한 번 더 묻는다(정책 23 §5-2). */
const CLOSED_FOR_MS = 14 * 24 * 60 * 60 * 1000;

export function closedRecently(raw: string | null, now: number): boolean {
  if (!raw) { return false; }
  // 예전 값("1", 세션 저장)은 오늘 닫은 것으로 본다.
  const at = raw === "1" ? now : Number(raw);
  return Number.isFinite(at) && now - at < CLOSED_FOR_MS;
}

function readClosedMark(): string | null {
  try {
    return window.localStorage.getItem(CLOSED_KEY) ?? window.sessionStorage.getItem(CLOSED_KEY);
  } catch (err) {
    console.warn("punisher: storage read failed", err);
    return null;
  }
}

function writeClosed(): void {
  try {
    window.localStorage.setItem(CLOSED_KEY, String(Date.now()));
  } catch (err) {
    console.warn("punisher: storage write failed", err);
  }
}

function clearClosed(): void {
  try {
    window.localStorage.removeItem(CLOSED_KEY);
    window.sessionStorage.removeItem(CLOSED_KEY);
  } catch (err) {
    console.warn("punisher: storage write failed", err);
  }
}

/**
 * 처음 설정 시트를 띄울지 — 설정을 받았고, 설명 수준이 빈 값("")이거나 전에 건너뛴 적이 있으며(기본 수준만 저장됨),
 * 최근 2주 안에 건너뛰지 않았을 때. 한 번 열리면 저장으로 수준이 채워져도(이미 앎 저장·실패 안내 중) 닫을 때까지 열어 둔다.
 */
export function useFirstSetupGate() {
  const settings = useLearningSettings();
  const level = settings.data?.settings.readerLevel;
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (level === undefined) { return; }
    const mark = readClosedMark();
    const skippedBefore = mark !== null;
    if ((level === "" || skippedBefore) && !closedRecently(mark, Date.now())) { setOpen(true); }
  }, [level]);

  /** 건너뛰기·×: 이 기기에서 2주 동안 닫는다(기본 수준 저장은 시트가 한다). */
  const skip = useCallback(() => {
    writeClosed();
    setOpen(false);
  }, []);
  /** 시작하기로 저장을 마쳤을 때 — 다시 묻지 않는다. */
  const finish = useCallback(() => {
    clearClosed();
    setOpen(false);
  }, []);

  return { open, skip, finish };
}

export interface FinishResult {
  ok: boolean;
  failedIds: string[];
}

/** 시작하기 — 설정 PUT → '설명 가능' 영역 깊이 1 개념을 기존 일괄 이미 앎으로 저장. 건너뛰기 — practitioner 만 저장. */
export function useFirstSetupSave() {
  const concepts = useConcepts();
  const save = useSaveLearningSettings();
  const batch = useBatchMarkKnown();

  const skip = useCallback(async () => {
    await save.mutateAsync({ readerLevel: DEFAULT_READER_LEVEL });
  }, [save]);

  const markKnown = useCallback(
    async (ids: string[]): Promise<MarkKnownProgress> => {
      if (ids.length === 0) { return { done: 0, total: 0, failedIds: [] }; }
      return batch.run(ids);
    },
    [batch],
  );

  const finish = useCallback(
    async (level: ChosenReaderLevel, familiarity: FamiliarityByArea): Promise<FinishResult> => {
      await save.mutateAsync({ readerLevel: level, areaFamiliarity: familiarity });
      const areas = explainableAreas(familiarity);
      if (areas.length === 0) { return { ok: true, failedIds: [] }; }
      const list = concepts.data ?? (await concepts.refetch()).data ?? [];
      const result = await markKnown(depthOneIdsForAreas(list, areas));
      return { ok: result.failedIds.length === 0, failedIds: result.failedIds };
    },
    [save, concepts, markKnown],
  );

  return {
    skip,
    finish,
    markKnown,
    saveError: save.error,
    savingSettings: save.isPending,
    markingKnown: batch.running,
    progress: batch.progress,
  };
}
