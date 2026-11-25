
import { Link } from "@/src/shared/lib/router";
import { useEffect, useId, useState } from "react";
import { Check, Circle, Lock, X } from "lucide-react";
import { BottomSheet, Button, ErrorState, IconButton, List, ListRow, ListSkeleton, Section } from "@/src/shared/ui/kit";
import {
  MasteryDots,
  STUDY_REQUEST_SOURCE_LABELS,
  USER_STATE_LABELS,
  UncheckedObjectives,
  areaLabel,
  useConceptDetail,
  useMastery,
  useNotes,
} from "@/src/entities/punisher";
import { KnownVerifyChoice, useKnownChoice } from "@/src/features/punisher/check-start";
import { ResolutionChoices, useLaunchInstant } from "@/src/features/punisher/instant-start";
import { lectureAnchorHref } from "@/src/entities/punisher";
import { describePunisherError, type ConceptLink, type InstantResponse, type MapRequest } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { ConceptLecturesSection } from "./ConceptLecturesSection";
import { ConceptQuestionsSection } from "./ConceptQuestionsSection";
import { messages } from "../i18n";
import { useT } from "@/src/shared/i18n";
import { isWitems } from "@/src/shared/config/runtime";
import { ConceptRecentReadsSection } from "./ConceptRecentReadsSection";
import { ConceptDismissAction } from "./ConceptDismissAction";

function LinkGroup({ title, links, onSelect }: { title: string; links: ConceptLink[]; onSelect: (id: string) => void }) {
  const t = useT(messages);
  if (links.length === 0) { return null; }
  return (
    <Section title={title} headingLevel={3}>
      <div className="flex flex-wrap gap-2">
        {links.map((l) => (
          <Button key={l.id} variant="secondary" size="sm" onClick={() => onSelect(l.id)}>
            {l.name}
            <span className="font-normal text-fg-muted">
              {USER_STATE_LABELS[l.userState] ?? l.userState}
              {l.dismissed ? ` · ${t.dismissedBadge}` : ""}
            </span>
          </Button>
        ))}
      </div>
    </Section>
  );
}

/** 뺐지만 아직 배우지 않은 선행 개념 — 이 개념을 잠그고 있는 이유. */
function isDismissedBlocker(p: ConceptLink): boolean {
  return Boolean(p.dismissed) && p.userState !== "known" && p.userState !== "completed";
}

function NotesGroup({ conceptId }: { conceptId: string }) {
  const notes = useNotes({ conceptId });
  const t = useT(messages);
  const items = notes.data ?? [];
  return (
    <Section title={t.notes} headingLevel={3}>
      {items.length === 0 ? (
        <p className="text-xs text-fg-muted">{t.notesEmpty}</p>
      ) : (
        <List>
          {items.map((n) =>
            n.lectureId ? (
              <ListRow
                key={n.id}
                href={lectureAnchorHref(n.lectureId, n.anchor)}
                linkAs={Link}
                title={n.body || n.quote}
              />
            ) : (
              <ListRow key={n.id} title={n.body || n.quote} />
            ),
          )}
        </List>
      )}
    </Section>
  );
}

interface ConceptDetailSheetProps {
  conceptId: string | null;
  /** 이 개념에 붙은 대기 학습 요청(지도 그래프에서) */
  requests?: MapRequest[];
  onClose: () => void;
  onSelect: (id: string) => void;
  /** sheet: 하단 시트(모바일)·오른쪽 시트(데스크톱). panel: 넓은 화면에서 캔버스 옆 열에 고정(시트 없이) */
  variant?: "sheet" | "panel";
}

/** 개념 상세: 모바일 하단 시트 / 데스크톱 오른쪽 시트 / 넓은 화면(xl) 캔버스 옆 고정 패널 (design §3.6 노드 클릭 → 개념 카드). */
export function ConceptDetailSheet({ conceptId, requests = [], onClose, onSelect, variant = "sheet" }: ConceptDetailSheetProps) {
  const titleId = useId();
  const t = useT(messages);
  const detail = useConceptDetail(conceptId ?? undefined);
  const mastery = useMastery(conceptId ?? undefined);
  const knownChoice = useKnownChoice();
  const known = knownChoice.known;
  const { launch, isPending, error } = useLaunchInstant();
  const [result, setResult] = useState<InstantResponse | null>(null);
  const resetKnown = known.reset;
  const closeKnown = knownChoice.close;

  // 다른 개념으로 넘어가면 이전 개념의 실행 결과를 지운다.
  useEffect(() => {
    setResult(null);
    resetKnown();
    closeKnown();
  }, [conceptId, resetKnown, closeKnown]);

  async function start(id: string) {
    try {
      setResult(await launch({ conceptId: id }));
    } catch {
      // 오류는 <InlineError error={error}> 가 표시한다.
      setResult(null);
    }
  }

  const c = detail.data;
  const learned = c?.userState === "known" || c?.userState === "completed";
  const description = c
    ? `${areaLabel(c.area)} · ${t.depth(c.depth)} · ${USER_STATE_LABELS[c.userState] ?? c.userState}${c.locked ? t.lockedSuffix : ""}`
    : undefined;

  const title = c?.name ?? conceptId ?? "";
  const conceptActions = c ? (
    <div className="flex w-full flex-wrap items-center justify-end gap-2">
      <Button variant="secondary" disabled={learned || known.isSuccess} loading={known.isPending} onClick={() => knownChoice.ask(c.id, c.userState)}>
        {known.isSuccess ? t.knownDone : t.known}
      </Button>
      <Button loading={isPending} onClick={() => start(c.id)}>
        {t.startLecture}
      </Button>
      <ConceptDismissAction conceptId={c.id} dismissed={Boolean(c.dismissed)} learned={learned} />
    </div>
  ) : null;
  const actions = knownChoice.asking ? <KnownVerifyChoice choice={knownChoice} variant="inline" /> : conceptActions;
  const body = (
    <>
      {detail.isLoading ? (
        <ListSkeleton rows={3} />
      ) : detail.error ? (
        <ErrorState compact message={describePunisherError(detail.error)} onRetry={() => detail.refetch()} />
      ) : c ? (
        <div className="flex flex-col gap-4">
          <div aria-live="polite">
            {/* '이미 앎' 저장 오류는 선택이 열려 있으면 그 안에서 보인다. */}
            <InlineError error={error ?? (knownChoice.asking ? null : known.error)} />
            {result && <ResolutionChoices response={result} disabled={isPending} onPick={start} />}
          </div>
          {c.locked && (
            <p className="flex items-center gap-1.5 text-sm text-fg-secondary">
              <Lock className="size-4" aria-hidden="true" />
              {c.prerequisites.some((link) => isDismissedBlocker(link)) ? t.dismissedPrereqHint : t.lockedHint}
            </p>
          )}
          {requests.length > 0 && (
            <div className="rounded-md bg-accent-subtle px-3 py-2 text-sm">
              <p className="font-semibold text-accent-text">{t.requestsTitle}</p>
              <ul className="mt-1 list-disc pl-5 text-fg">
                {requests.map((r) => (
                  <li key={r.id}>
                    {r.query}
                    <span className="text-xs text-fg-muted"> · {STUDY_REQUEST_SOURCE_LABELS[r.source] ?? r.source}</span>
                  </li>
                ))}
              </ul>
              <p className="mt-1 text-xs text-fg-secondary">{t.requestsHint}</p>
            </div>
          )}
          {c.summary && <p className="text-sm text-fg-secondary">{c.summary}</p>}
          <Section title={t.objectives} headingLevel={3}>
            <UncheckedObjectives count={mastery.data?.uncheckedCount} />
            <ul role="list" className="flex flex-col gap-1.5 text-sm">
              {c.objectives.map((o) => {
                const done = o.userState === "completed" || o.userState === "known";
                const m = mastery.data?.objectives.find((x) => x.objectiveId === o.id);
                return (
                  <li key={o.id} className="flex items-start gap-2">
                    {done ? (
                      <Check className="mt-0.5 size-4 shrink-0 text-success" aria-hidden="true" />
                    ) : (
                      <Circle className="mt-0.5 size-4 shrink-0 text-fg-muted" aria-hidden="true" />
                    )}
                    <span className="min-w-0 flex-1 text-fg">
                      <span className="sr-only">{done ? t.objectiveDone : t.objectiveTodo}</span>
                      {o.text} <span className="text-xs text-fg-muted">{t.objectiveDepth(o.depth)}</span>
                    </span>
                    {m && <MasteryDots level={m.level} lastGrader={m.lastGrader} className="mt-1" />}
                  </li>
                );
              })}
            </ul>
          </Section>
          <LinkGroup title={t.prerequisites} links={c.prerequisites} onSelect={onSelect} />
          <LinkGroup title={t.dependents} links={c.dependents} onSelect={onSelect} />
          <LinkGroup title={t.related} links={c.related} onSelect={onSelect} />
          <ConceptLecturesSection conceptId={c.id} />
          <ConceptQuestionsSection conceptId={c.id} />
          <NotesGroup conceptId={c.id} />
          {/* Witems 안에서만: 이 개념이 나온 최근에 연 RSS 글 */}
          {isWitems() ? <ConceptRecentReadsSection conceptId={c.id} /> : null}
        </div>
      ) : null}
    </>
  );

  if (variant === "panel") {
    if (!conceptId) { return null; }
    return (
      <section aria-labelledby={titleId} className="flex flex-col gap-4 rounded-xl border border-line bg-canvas-elevated p-4 shadow-xs">
        <header className="flex items-start gap-2">
          <div className="min-w-0 flex-1">
            <h2 id={titleId} className="break-keep text-lg font-semibold text-fg">{title}</h2>
            {description && <p className="mt-0.5 text-xs text-fg-muted">{description}</p>}
          </div>
          <IconButton aria-label={t.closeDetail} onClick={onClose} className="-mr-2 -mt-1">
            <X aria-hidden="true" />
          </IconButton>
        </header>
        {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
        {body}
      </section>
    );
  }

  return (
    <BottomSheet open={Boolean(conceptId)} onClose={onClose} title={title} description={description} desktop="side" footer={actions}>
      {body}
    </BottomSheet>
  );
}
