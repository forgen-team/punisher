import { useState } from "react";
import { GraduationCap, Map as MapIcon, Sparkles } from "lucide-react";
import { IconButton, List, ListRow, ProgressCircle, Section } from "@forgen-team/ui";
import { USER_STATE_LABELS_BY_LOCALE, areaLabel } from "@/src/entities/punisher";
import { useLaunchInstant } from "@/src/features/punisher/instant-start";
import { useLocale, useT } from "@/src/shared/i18n";
import { InlineError } from "@/src/shared/ui/punisher";
import { messages } from "../i18n";
import type { InquiryConcept, UserConceptState } from "@/src/shared/api/punisher";

function mapHref(id: string): string {
  return `/map?concept=${encodeURIComponent(id)}`;
}

/**
 * 해설에 나온 개념 — 지도에 있던 개념(관련)과 해설이 지도에 새로 제안한 개념. 이 질문은 해당 노드에 '내 질문'으로
 * 붙어서, 나중에 지도에서 그 개념을 보면 함께 보인다. 오른쪽 버튼은 그 개념의 일반 강의를 연다.
 */
export function InquiryConcepts({ concepts }: { concepts: InquiryConcept[] }) {
  const locale = useLocale();
  const t = useT(messages);
  const { launch, isPending, error } = useLaunchInstant();
  const [launching, setLaunching] = useState("");
  const related = concepts.filter((c) => c.kind === "related");
  const proposed = concepts.filter((c) => c.kind === "proposed");
  if (concepts.length === 0) { return null; }

  const row = (c: InquiryConcept) => (
    <ListRow
      key={c.id}
      href={mapHref(c.id)}
      leading={c.kind === "proposed" ? <Sparkles aria-hidden="true" /> : <MapIcon aria-hidden="true" />}
      title={c.name}
      meta={c.kind === "proposed" ? t.proposedConcept(areaLabel(c.area)) : `${areaLabel(c.area)} · ${USER_STATE_LABELS_BY_LOCALE[locale][c.userState as UserConceptState] ?? c.userState}`}
      trailing={
        <IconButton
          size="sm"
          aria-label={t.openLecture(c.name)}
          title={t.openLectureTitle}
          disabled={isPending}
          onClick={() => {
            setLaunching(c.id);
            void launch({ conceptId: c.id }).catch(() => undefined);
          }}
          icon={isPending && launching === c.id ? <ProgressCircle size="24" tone="inherit" aria-hidden /> : <GraduationCap aria-hidden="true" />}
        />
      }
    />
  );

  return (
    <div className="flex flex-col gap-6">
      {related.length > 0 && (
        <Section title={t.related} description={t.relatedDescription} headingLevel={3}>
          <List>{related.map((c) => row(c))}</List>
        </Section>
      )}
      {proposed.length > 0 && (
        <Section title={t.proposed} description={t.proposedDescription} headingLevel={3}>
          <List>{proposed.map((c) => row(c))}</List>
        </Section>
      )}
      <InlineError error={error} />
    </div>
  );
}
