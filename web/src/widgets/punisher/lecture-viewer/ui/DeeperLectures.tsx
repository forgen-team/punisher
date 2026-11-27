
import { Link } from "@/src/shared/lib/router";
import { forwardRef } from "react";
import { ArrowLeft, Layers } from "lucide-react";
import { List, ListRow, Section } from "@forgen-team/ui";
import { isReadThrough, readThroughLabel } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { lectureAnchorHref } from "@/src/entities/punisher";
import type { LinkedLecture } from "@/src/shared/api/punisher";
import { formatDate, getLocale, useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { versionRows, type CurrentVersion } from "../lib/versionRows";

type T = typeof messages.ko;

function progressLabel(l: LinkedLecture, t: T): string {
  if (l.completedAt) { return t.progressDone; }
  if (isReadThrough(l.percent, false)) { return readThroughLabel(getLocale()); }
  if (l.percent > 0) { return t.percentRead(l.percent); }
  return t.unread;
}

/** 심화 강의 상단: "← 원 강의로" (더 깊게 누른 위치로 돌아간다). */
export function ParentLectureLink({ parent }: { parent: LinkedLecture }) {
  const t = useT(messages);
  const href = lectureAnchorHref(parent.id, parent.anchor);
  return (
    <div className="border-b border-line py-1 text-sm">
      <Link href={href} className="inline-flex min-h-touch min-w-0 max-w-full items-center gap-1 text-accent-text">
        <ArrowLeft className="size-4 shrink-0" aria-hidden="true" />
        <span className="truncate">
          {parent.linkKind === "regenerated" ? t.backToOriginalVersion : t.backToParent}: {parent.conceptName}
          {parent.title && parent.title !== parent.conceptName ? ` · ${parent.title}` : ""}
        </span>
      </Link>
    </div>
  );
}

function sectionCopy(kind: "deeper" | "regenerated", t: T) {
  return kind === "deeper"
    ? { title: t.deeperTitle, description: t.deeperDescription }
    : { title: t.regeneratedTitle, description: t.regeneratedDescription };
}

interface LinkedLecturesSectionProps {
  lectures: LinkedLecture[];
  kind: "deeper" | "regenerated";
  /** regenerated: 지금 보는 판(목록에 '지금 보는 판'으로 같이 보인다) */
  current?: CurrentVersion;
}

/** 원 강의 맨 아래: "더 깊게" 본 강의 / 다시 만든 강의(개념 · 제목 · 생성일 · 진도). */
export const DeeperLecturesSection = forwardRef<HTMLElement, LinkedLecturesSectionProps>(function DeeperLecturesSection(
  { lectures, kind, current },
  ref,
) {
  const t = useT(messages);
  const locale = useLocale();
  if (lectures.length === 0) { return null; }
  const copy = sectionCopy(kind, t);
  const day = (iso: string) => formatDate(new Date(iso), locale, { dateStyle: "medium" });
  if (kind === "regenerated") {
    // 같은 제목이 반복되던 줄: 주 줄은 '판 n · 만든 날 · 프롬프트 버전', 제목은 지금 보는 판과 다를 때만.
    return (
      <section ref={ref} id={`${kind}-lectures`} tabIndex={-1} className="mt-10 scroll-mt-28 outline-none">
        <Section title={copy.title} description={copy.description}>
          <List>
            {versionRows(lectures, current).map((r) => {
              const meta = [r.distinctTitle, r.current ? t.currentVersion : r.linked ? progressLabel(r.linked, t) : ""].filter(Boolean).join(" · ");
              const title = t.versionLine(r.version, day(r.createdAt), r.promptVersion);
              return r.current ? (
                <ListRow key={r.id} selected leading={<Layers aria-hidden="true" />} title={title} meta={meta} />
              ) : (
                <ListRow key={r.id} href={lectureHref(r.id)} leading={<Layers aria-hidden="true" />} title={title} meta={meta} />
              );
            })}
          </List>
        </Section>
      </section>
    );
  }
  return (
    <section ref={ref} id={`${kind}-lectures`} tabIndex={-1} className="mt-10 scroll-mt-28 outline-none">
      <Section title={copy.title} description={copy.description}>
        <List>
          {lectures.map((l) => (
            <ListRow
              key={l.id}
              href={lectureHref(l.id)}
              leading={<Layers aria-hidden="true" />}
              title={l.conceptName || l.title}
              meta={[
                l.title && l.title !== l.conceptName ? l.title : "",
                t.createdOn(formatDate(new Date(l.createdAt), locale, { dateStyle: "medium" })),
                progressLabel(l, t),
              ]
                .filter(Boolean)
                .join(" · ")}
            />
          ))}
        </List>
      </Section>
    </section>
  );
});
