
import { ErrorState, ListSkeleton, Section } from "@forgen-team/ui";
import { describeCheckError, useMasteryMetrics } from "@/src/entities/punisher";
import { metricRows } from "../lib/checkHistory";
import { METRIC_GRADER_LABELS, messages } from "../i18n";
import { METRIC_GRADER_ORDER } from "./constants";
import { useLocale, useT } from "@/src/shared/i18n";

/** 기억 유지 지표 — 간격별 · 채점 주체별 첫 시도 정답률(최근 n주). 시도 없는 칸은 '—'. */
export function RetentionMetrics() {
  const locale = useLocale();
  const t = useT(messages);
  const metrics = useMasteryMetrics();
  const weeks = metrics.data?.weeks;
  return (
    <Section title={t.retention} description={weeks ? t.recentWeeks(weeks) : undefined}>
      {renderBody()}
    </Section>
  );

  function renderBody() {
    if (metrics.isLoading) { return <ListSkeleton rows={3} />; }
    if (metrics.error || !metrics.data) {
      return <ErrorState title={describeCheckError(metrics.error)} onRetry={() => metrics.refetch()} />;
    }
    const rows = metricRows(metrics.data.buckets);
    return (
      <table className="w-full max-w-xl table-fixed border-collapse text-sm">
        <thead>
          <tr className="border-b border-line text-left text-xs text-fg-muted">
            <th scope="col" className="py-2 pr-3 font-medium">{t.interval}</th>
            {METRIC_GRADER_ORDER.map((g) => (
              <th key={g} scope="col" className="py-2 pr-3 font-medium">
                {METRIC_GRADER_LABELS[locale][g]}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.intervalDays} className="border-b border-line last:border-b-0">
              <th scope="row" className="py-2 pr-3 text-left font-normal text-fg-secondary">
                {t.intervalDays(row.intervalDays)}
              </th>
              {METRIC_GRADER_ORDER.map((g) => {
                const cell = row.cells[g];
                return (
                  <td key={g} className="py-2 pr-3 tabular-nums text-fg">
                    {cell.rate}
                    {cell.count && <span className="ml-1.5 text-xs text-fg-muted">{cell.count}</span>}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    );
  }
}
