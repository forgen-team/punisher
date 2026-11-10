
import { Link } from "@/src/shared/lib/router";
import { PlayCircle } from "lucide-react";
import { BottomSheet, Button, ListRow, buttonVariants } from "@/src/shared/ui/kit";
import { RECOMMENDATION_KIND_LABELS, areaLabel } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import type { Recommendation } from "@/src/shared/api/punisher";

function mapHref(conceptId: string): string {
  return `/map?concept=${encodeURIComponent(conceptId)}`;
}

/** 강의 제목이 생성마다 달라지므로 목록에서는 개념 이름을 앞세우고 강의 제목은 보조로 둔다. */
export function lectureSubtitle(item: Pick<Recommendation, "title" | "conceptName">): string {
  return item.title && item.title !== item.conceptName ? item.title : "";
}

interface RecommendationRowProps {
  item: Recommendation;
  onOpen: (item: Recommendation) => void;
}

/** 추천 행: 제목 / 영역 · 약 N분 / 왜 오늘(한 줄). 상세는 시트로. */
export function RecommendationRow({ item, onOpen }: RecommendationRowProps) {
  return (
    <ListRow
      onClick={() => onOpen(item)}
      title={item.conceptName || item.title}
      meta={
        <>
          {lectureSubtitle(item) && <span className="block truncate text-fg-secondary">{lectureSubtitle(item)}</span>}
          {areaLabel(item.area)} · 약 {item.estimatedMinutes}분
          {item.reason && <span className="block truncate text-fg-secondary">{item.reason}</span>}
        </>
      }
    />
  );
}

interface RecommendationSheetProps {
  item: Recommendation | null;
  starting: boolean;
  onClose: () => void;
  onStart: (conceptId: string) => void;
}

/** 추천 상세: 왜 오늘 · 이해하게 될 실무 상황 · 분량 + 강의 열기/시작. */
export function RecommendationSheet({ item, starting, onClose, onStart }: RecommendationSheetProps) {
  const title = item ? item.conceptName || item.title : "";
  return (
    <BottomSheet
      open={Boolean(item)}
      onClose={onClose}
      title={title}
      description={
        item
          ? `${RECOMMENDATION_KIND_LABELS[item.kind] ?? item.kind} · ${areaLabel(item.area)} · 깊이 ${item.depth} · 약 ${item.estimatedMinutes}분`
          : undefined
      }
      footer={
        item ? (
          <>
            <Link href={mapHref(item.conceptId)} className={buttonVariants({ variant: "ghost" })}>
              지도에서 보기
            </Link>
            {item.lectureId ? (
              <Link href={lectureHref(item.lectureId)} className={buttonVariants({ variant: "primary" })}>
                <PlayCircle className="size-4" aria-hidden="true" />
                강의 열기
              </Link>
            ) : (
              <Button loading={starting} onClick={() => onStart(item.conceptId)}>
                강의 시작
              </Button>
            )}
          </>
        ) : null
      }
    >
      {item && (
        <dl className="flex flex-col gap-3 text-sm">
          {item.title && item.title !== item.conceptName && (
            <div>
              <dt className="text-xs font-medium text-fg-muted">강의 제목</dt>
              <dd className="text-fg">{item.title}</dd>
            </div>
          )}
          <div>
            <dt className="text-xs font-medium text-fg-muted">왜 오늘</dt>
            <dd className="text-fg">{item.reason}</dd>
          </div>
          {item.practicalSituation && (
            <div>
              <dt className="text-xs font-medium text-fg-muted">이해하게 될 실무 상황</dt>
              <dd className="text-fg">{item.practicalSituation}</dd>
            </div>
          )}
        </dl>
      )}
    </BottomSheet>
  );
}
