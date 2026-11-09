
import { Link } from "@/src/shared/lib/router";
import { ChevronRight } from "lucide-react";
import type { ConceptDetail, Lecture, LectureContent } from "@/src/shared/api/punisher";
import { LectureReviewBadges, USER_STATE_LABELS, areaLabel } from "@/src/entities/punisher";

function mapHref(conceptId: string): string {
  return `/map?concept=${encodeURIComponent(conceptId)}`;
}

interface LectureHeaderProps {
  /** 즉석 생성 스트리밍 중에는 강의 레코드가 없다 */
  lecture?: Pick<Lecture, "kind" | "status" | "flagged" | "flagReason" | "depth">;
  content: Pick<LectureContent, "title" | "estimatedMinutes" | "objectiveIds" | "newScope" | "prerequisitesReferenced">;
  concept?: ConceptDetail;
  /** 제목 대체 문구 (스트리밍 시작 전) */
  fallbackTitle?: string;
}

/**
 * 개념 이름(고정) → 제목(h1) + 메타 한 줄(영역 · 깊이 · 약 N분 · 내 상태) + 배지 최대 1개.
 * 학습 목표 · 새로 다루는 범위 · 이전에 배운 개념은 접힌 상태로 둬 본문이 첫 화면에 보이게 한다.
 */
export function LectureHeader({ lecture, content, concept, fallbackTitle }: LectureHeaderProps) {
  const objectiveText = new Map((concept?.objectives ?? []).map((o) => [o.id, o.text]));
  const prereqNames = new Map((concept?.prerequisites ?? []).map((p) => [p.id, p.name]));
  const meta = [
    concept ? areaLabel(concept.area) : "",
    lecture ? `깊이 ${lecture.depth}` : "",
    content.estimatedMinutes ? `약 ${content.estimatedMinutes}분` : "",
    concept ? (USER_STATE_LABELS[concept.userState] ?? concept.userState) : "",
  ].filter(Boolean);
  const hasScope =
    content.objectiveIds.length > 0 || content.newScope.length > 0 || content.prerequisitesReferenced.length > 0;

  return (
    <header className="flex flex-col gap-2 pt-4">
      {concept && (
        // 강의 제목은 생성마다 달라지므로, 무엇을 배우는지(개념 이름)를 제목 위에 고정해 둔다.
        <p className="text-sm font-semibold text-accent-text">{concept.name}</p>
      )}
      <h1 className="text-xl font-semibold text-fg md:text-2xl">{content.title || fallbackTitle}</h1>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-fg-muted">
        {concept ? (
          <Link href={mapHref(concept.id)} className="hover:text-fg hover:underline">
            {meta.join(" · ")}
          </Link>
        ) : (
          <span>{meta.join(" · ")}</span>
        )}
        {lecture && <LectureReviewBadges lecture={lecture} />}
      </div>
      {hasScope && (
        <details className="group border-y border-line text-sm">
          <summary className="flex min-h-touch cursor-pointer list-none items-center gap-1 text-fg-secondary [&::-webkit-details-marker]:hidden">
            <ChevronRight className="size-4 transition-transform group-open:rotate-90" aria-hidden="true" />
            학습 목표 · 새로 다루는 범위
          </summary>
          <div className="flex flex-col gap-3 pb-3 pl-5">
            {content.objectiveIds.length > 0 && (
              <div>
                <p className="text-xs font-semibold text-fg-secondary">학습 목표</p>
                <ul className="mt-1 list-disc space-y-0.5 pl-5 text-fg">
                  {content.objectiveIds.map((id) => (
                    <li key={id}>{objectiveText.get(id) ?? id}</li>
                  ))}
                </ul>
              </div>
            )}
            {content.newScope.length > 0 && (
              <div>
                <p className="text-xs font-semibold text-fg-secondary">이번에 새로 다루는 범위</p>
                <ul className="mt-1 list-disc space-y-0.5 pl-5 text-fg">
                  {content.newScope.map((line, index) => (
                    <li key={index}>{line}</li>
                  ))}
                </ul>
              </div>
            )}
            {content.prerequisitesReferenced.length > 0 && (
              <p className="text-fg-secondary">
                이전에 배운 개념:{" "}
                {content.prerequisitesReferenced.map((id, index) => (
                  <span key={id}>
                    {index > 0 && ", "}
                    <Link href={mapHref(id)} className="text-accent-text underline underline-offset-2">
                      {prereqNames.get(id) ?? id}
                    </Link>
                  </span>
                ))}
              </p>
            )}
          </div>
        </details>
      )}
    </header>
  );
}
