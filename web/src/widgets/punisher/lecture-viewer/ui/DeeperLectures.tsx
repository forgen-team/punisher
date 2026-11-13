
import { Link } from "@/src/shared/lib/router";
import { forwardRef } from "react";
import { ArrowLeft, Layers } from "lucide-react";
import { List, ListRow, Section } from "@/src/shared/ui/kit";
import { READ_THROUGH_LABEL, isReadThrough } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import type { LinkedLecture } from "@/src/shared/api/punisher";

const dateFormat = new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium" });

function progressLabel(l: LinkedLecture): string {
  if (l.completedAt) { return "완료"; }
  if (isReadThrough(l.percent, false)) { return READ_THROUGH_LABEL; }
  if (l.percent > 0) { return `${l.percent}% 읽음`; }
  return "아직 안 읽음";
}

/** 심화 강의 상단: "← 원 강의로" (더 깊게 누른 위치로 돌아간다). */
export function ParentLectureLink({ parent }: { parent: LinkedLecture }) {
  const href = `${lectureHref(parent.id)}${parent.anchor ? `#pa-${parent.anchor}` : ""}`;
  return (
    <div className="border-b border-line py-1 text-sm">
      <Link href={href} className="inline-flex min-h-touch min-w-0 max-w-full items-center gap-1 text-accent-text">
        <ArrowLeft className="size-4 shrink-0" aria-hidden="true" />
        <span className="truncate">
          {parent.linkKind === "regenerated" ? "원래 버전으로" : "원 강의로"}: {parent.conceptName}
          {parent.title && parent.title !== parent.conceptName ? ` · ${parent.title}` : ""}
        </span>
      </Link>
    </div>
  );
}

const SECTION_COPY = {
  deeper: { title: "이 강의에서 더 깊게 본 강의", description: "용어의 '이 개념 더 깊게'로 연 강의예요. 언제든 다시 볼 수 있어요." },
  regenerated: { title: "새 프롬프트로 다시 만든 강의", description: "내 프롬프트로 같은 목표를 다시 만든 버전이에요." },
} as const;

interface LinkedLecturesSectionProps {
  lectures: LinkedLecture[];
  kind: keyof typeof SECTION_COPY;
}

/** 원 강의 맨 아래: "더 깊게" 본 강의 / 다시 만든 강의(개념 · 제목 · 생성일 · 진도). */
export const DeeperLecturesSection = forwardRef<HTMLElement, LinkedLecturesSectionProps>(function DeeperLecturesSection(
  { lectures, kind },
  ref,
) {
  if (lectures.length === 0) { return null; }
  return (
    <section ref={ref} id={`${kind}-lectures`} tabIndex={-1} className="mt-10 scroll-mt-28 outline-none">
      <Section title={SECTION_COPY[kind].title} description={SECTION_COPY[kind].description}>
        <List>
          {lectures.map((l) => (
            <ListRow
              key={l.id}
              href={lectureHref(l.id)}
              linkAs={Link}
              leading={<Layers aria-hidden="true" />}
              title={l.conceptName || l.title}
              meta={[
                l.title && l.title !== l.conceptName ? l.title : "",
                `${dateFormat.format(new Date(l.createdAt))} 생성`,
                progressLabel(l),
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
