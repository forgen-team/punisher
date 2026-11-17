import { CHECK_KIND_LABELS_BY_LOCALE } from "@/src/entities/punisher";
import type { CheckHistoryItem, Misconception, RetentionBucket } from "@/src/shared/api/punisher";
import { translate, type Locale } from "@/src/shared/i18n";
import { HISTORY_STATUS_LABELS, messages } from "../i18n";
import { historyDay } from "./historyDate";


/** 세션 날짜 — 복습은 그 KST 날짜(reviewDay), 확인은 시작 시각. */
export function sessionDate(item: CheckHistoryItem, locale: Locale): string {
  const source = item.reviewDay ? `${item.reviewDay}T00:00:00+09:00` : item.createdAt;
  return historyDay(source, locale);
}

/** 고친 오개념 한 줄 메타 — 개념 · 고친 날짜. */
export function resolvedMeta(m: Pick<Misconception, "conceptName" | "resolvedAt">, locale: Locale): string {
  const day = m.resolvedAt ? historyDay(m.resolvedAt, locale) : "";
  return [m.conceptName, day].filter(Boolean).join(" · ");
}

/** 기록 한 줄 제목 — "확인 · 개념 · 강의" / "복습". */
export function historyTitle(item: CheckHistoryItem, locale: Locale): string {
  const subject = [item.conceptName, item.lectureTitle].filter((v, index, all) => v && all.indexOf(v) === index);
  return [CHECK_KIND_LABELS_BY_LOCALE[locale][item.kind], ...subject].join(" · ");
}

/** 기록 한 줄 메타 — 날짜 · n/N · 혼자 맞힘 k (· 진행 중/그만둠). */
export function historyMeta(item: CheckHistoryItem, locale: Locale): string {
  return [
    sessionDate(item, locale),
    `${item.doneCount}/${item.itemCount}`,
    translate(messages, locale).unaided(item.unaidedCorrect),
    HISTORY_STATUS_LABELS[locale][item.status] ?? item.status,
  ]
    .filter(Boolean)
    .join(" · ");
}

export interface MetricCell {
  /** 정답률 "67%" — 시도가 없으면 "—" */
  rate: string;
  /** "2/3" — 시도가 없으면 "" */
  count: string;
}

export interface MetricRow {
  intervalDays: number;
  cells: Record<"llm" | "self", MetricCell>;
}

const EMPTY_CELL: MetricCell = { rate: "—", count: "" };

function cellOf(b: RetentionBucket): MetricCell {
  if (b.rate === null || b.attempts === 0) { return EMPTY_CELL; }
  return { rate: `${Math.round(b.rate * 100)}%`, count: `${b.correct}/${b.attempts}` };
}

/** 지표 표 — 간격(일) 오름차순 줄 × 채점 주체 칸. 없는 칸·rate null 은 '—'. */
export function metricRows(buckets: RetentionBucket[]): MetricRow[] {
  const byInterval = new Map<number, MetricRow>();
  for (const b of buckets) {
    const row = byInterval.get(b.intervalDays) ?? { intervalDays: b.intervalDays, cells: { llm: EMPTY_CELL, self: EMPTY_CELL } };
    row.cells = { ...row.cells, [b.grader]: cellOf(b) };
    byInterval.set(b.intervalDays, row);
  }
  return [...byInterval.values()].sort((a, b) => a.intervalDays - b.intervalDays);
}
