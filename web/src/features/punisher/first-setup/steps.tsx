import { cn } from "@/src/shared/ui/kit";
import { READER_LEVEL_LABELS_BY_LOCALE, areaLabel } from "@/src/entities/punisher";
import { FAMILIARITY_LEVELS, FAMILIARITY_AREAS, READER_LEVELS, type ChosenReaderLevel, type Familiarity, type FamiliarityArea } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { FAMILIARITY_LABELS, LEVEL_DESCRIPTIONS, messages } from "./i18n";
import { explainableAreas, startsWithFoundations, type FamiliarityByArea } from "./useFirstSetup";

/** 1단계 — 개발 경험 4개 중 하나(네이티브 라디오: 화살표 키로 이동). */
export function LevelStep({ value, onChange, disabled }: { value: ChosenReaderLevel | null; onChange: (level: ChosenReaderLevel) => void; disabled?: boolean }) {
  const locale = useLocale();
  const t = useT(messages);
  const descriptions = useT(LEVEL_DESCRIPTIONS);
  return (
    <div role="radiogroup" aria-label={t.levelGroup} className="flex flex-col gap-2">
      {READER_LEVELS.map((level) => {
        const selected = value === level;
        return (
          <label
            key={level}
            className={cn(
              "flex min-h-touch cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
              selected ? "border-accent bg-accent-subtle" : "border-line hover:bg-canvas-subtle",
            )}
          >
            <input
              type="radio"
              name="first-setup-level"
              className="size-5 shrink-0 accent-accent-solid"
              value={level}
              checked={selected}
              disabled={disabled}
              onChange={() => onChange(level)}
            />
            <span className="min-w-0 flex-1">
              <span className={cn("block text-sm font-medium", selected ? "text-accent-text" : "text-fg")}>
                {READER_LEVEL_LABELS_BY_LOCALE[locale][level]}
              </span>
              <span className="block text-xs text-fg-muted">{descriptions[level]}</span>
            </span>
          </label>
        );
      })}
    </div>
  );
}

/** 2단계 — 영역마다 처음 · 들어봄 · 설명 가능. */
export function AreasStep({ value, onChange }: { value: FamiliarityByArea; onChange: (area: FamiliarityArea, familiarity: Familiarity) => void }) {
  const locale = useLocale();
  const labels = useT(FAMILIARITY_LABELS);
  const items = FAMILIARITY_LEVELS.map((f) => ({ value: f, label: labels[f] }));
  return (
    <ul role="list" className="border-y border-line">
      {FAMILIARITY_AREAS.map((area) => {
        const name = areaLabel(area, locale);
        return (
          <li key={area} className="flex min-h-touch flex-wrap items-center justify-between gap-x-3 gap-y-1 border-b border-line py-2 last:border-b-0">
            <span className="min-w-0 text-sm font-medium text-fg">{name}</span>
            <FamiliarityRadios area={area} name={name} items={items} value={value[area]} onChange={(f) => onChange(area, f)} />
          </li>
        );
      })}
    </ul>
  );
}

interface FamiliarityRadiosProps {
  area: FamiliarityArea;
  name: string;
  items: Array<{ value: Familiarity; label: string }>;
  value: Familiarity;
  onChange: (familiarity: Familiarity) => void;
}

/** 영역 하나의 처음 · 들어봄 · 설명 가능 — 보기는 알약 트랙, 의미는 네이티브 라디오(화살표 키로 이동). */
function FamiliarityRadios({ area, name, items, value, onChange }: FamiliarityRadiosProps) {
  return (
    <div role="radiogroup" aria-label={name} className="flex w-fit max-w-full gap-0.5 rounded-lg border border-line-subtle bg-canvas-subtle p-0.5">
      {items.map((item) => {
        const selected = item.value === value;
        return (
          <label
            key={item.value}
            className={cn(
              "relative inline-flex h-9 cursor-pointer items-center justify-center whitespace-nowrap rounded-md px-3 text-sm tracking-ui",
              "before:absolute before:inset-x-0 before:-inset-y-1 before:content-['']",
              "focus-within:ring-2 focus-within:ring-inset focus-within:ring-focus",
              selected ? "bg-canvas-selected font-medium text-fg shadow-sm" : "text-fg-muted hover:text-fg",
            )}
          >
            <input
              type="radio"
              className="sr-only"
              name={`first-setup-area-${area}`}
              value={item.value}
              checked={selected}
              onChange={() => onChange(item.value)}
            />
            {item.label}
          </label>
        );
      })}
    </div>
  );
}

/** 3단계 — 고른 것 요약. 0단계 기초 트랙은 아직 안내만(설계 ②에서 지도에 반영). */
export function SummaryStep({ level, familiarity }: { level: ChosenReaderLevel; familiarity: FamiliarityByArea }) {
  const locale = useLocale();
  const t = useT(messages);
  const skipped = explainableAreas(familiarity).map((area) => areaLabel(area, locale));
  const rows: Array<{ label: string; value: string }> = [
    { label: t.summaryLevel, value: READER_LEVEL_LABELS_BY_LOCALE[locale][level] },
    { label: t.summarySkipped, value: skipped.length > 0 ? t.skippedAreas(skipped.join(", ")) : t.skippedNone },
    { label: t.summaryFoundations, value: startsWithFoundations(level) ? t.foundationsStart : t.foundationsHidden },
  ];
  return (
    <div className="flex flex-col gap-3">
      <dl className="border-y border-line">
        {rows.map((row) => (
          <div key={row.label} className="flex min-h-touch items-center justify-between gap-3 border-b border-line py-2 last:border-b-0">
            <dt className="shrink-0 text-sm text-fg-secondary">{row.label}</dt>
            <dd className="min-w-0 text-right text-sm font-medium text-fg">{row.value}</dd>
          </div>
        ))}
      </dl>
      <p className="text-xs text-fg-muted">{t.note}</p>
    </div>
  );
}
