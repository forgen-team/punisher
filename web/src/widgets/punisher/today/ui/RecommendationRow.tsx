
import { Link } from "@/src/shared/lib/router";
import { useEffect } from "react";
import { BookOpen, PlayCircle } from "lucide-react";
import { BottomSheet, Button, ListRow, buttonVariants, notifyUndo } from "@/src/shared/ui/kit";
import { RECOMMENDATION_KIND_LABELS_BY_LOCALE, areaLabel, recommendationReason, useConceptDismiss } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import type { Recommendation } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

export function mapHref(conceptId: string): string {
  return `/map?concept=${encodeURIComponent(conceptId)}`;
}

/** 강의 제목이 생성마다 달라지므로 목록에서는 개념 이름을 앞세우고 강의 제목은 보조로 둔다. */
export function lectureSubtitle(item: Pick<Recommendation, "title" | "conceptName">): string {
  return item.title && item.title !== item.conceptName ? item.title : "";
}

/** 추천 상세 시트 주소(오늘 화면의 `?rec=<개념>`) — 화면=주소, 새 탭으로 열어도 같은 시트. */
export const RECOMMENDATION_PARAM = "rec";

export function recommendationHref(conceptId: string): string {
  return `/?${RECOMMENDATION_PARAM}=${encodeURIComponent(conceptId)}`;
}

interface RecommendationRowProps {
  item: Recommendation;
}

/** 추천 행: 제목 / 영역 · 약 N분 / 왜 오늘(두 줄까지). 상세는 시트로 — 행은 그 시트 주소로 가는 링크. */
export function RecommendationRow({ item }: RecommendationRowProps) {
  const t = useT(messages);
  return (
    <ListRow
      href={recommendationHref(item.conceptId)}
      linkAs={Link}
      leading={<BookOpen aria-hidden />}
      title={item.conceptName || item.title}
      meta={
        <>
          {lectureSubtitle(item) && <span className="block truncate text-fg-secondary">{lectureSubtitle(item)}</span>}
          {t.areaMinutes(areaLabel(item.area), item.estimatedMinutes)}
          {recommendationReason(item) && <span className="line-clamp-2 whitespace-normal break-keep text-fg-secondary">{recommendationReason(item)}</span>}
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

/** 추천 상세: 왜 오늘 · 이해하게 될 실무 상황 · 분량 + 강의 열기/시작 · 필요 없음(빼면 닫히고 목록이 새로 읽힌다). */
export function RecommendationSheet({ item, starting, onClose, onStart }: RecommendationSheetProps) {
  const locale = useLocale();
  const t = useT(messages);
  const dismiss = useConceptDismiss();
  const conceptId = item?.conceptId;
  const { reset } = dismiss;
  // 다른 추천을 열면 앞 추천의 '필요 없음' 오류·진행 표시를 지운다.
  useEffect(() => {
    reset();
  }, [conceptId, reset]);
  // '필요 없음'은 되돌릴 수 있어 확인 없이 빼고 되돌리기 알림을 띄운다.
  const dismissItem = (conceptId: string) =>
    dismiss.mutate({ conceptId, dismissed: true }, {
      onSuccess: () => {
        dismiss.reset();
        onClose();
        notifyUndo(t.dismissedToast, () => dismiss.mutateAsync({ conceptId, dismissed: false }));
      },
    });
  const title = item ? item.conceptName || item.title : "";
  return (
    <BottomSheet
      open={Boolean(item)}
      onClose={onClose}
      title={title}
      description={
        item
          ? t.sheetDescription(RECOMMENDATION_KIND_LABELS_BY_LOCALE[locale][item.kind] ?? item.kind, areaLabel(item.area), item.depth, item.estimatedMinutes)
          : undefined
      }
      footer={
        item ? (
          // 390px 폭에서 버튼 셋이 넘치지 않게 줄을 바꿔 오른쪽 정렬한다.
          <div className="flex w-full flex-wrap items-center justify-end gap-2">
            <Button variant="ghost" loading={dismiss.isPending} onClick={() => dismissItem(item.conceptId)}>
              {t.dismiss}
            </Button>
            <Link href={mapHref(item.conceptId)} className={buttonVariants({ variant: "ghost" })}>
              {t.viewOnMap}
            </Link>
            {item.lectureId ? (
              <Link href={lectureHref(item.lectureId)} className={buttonVariants({ variant: "primary" })}>
                <PlayCircle className="size-4" aria-hidden="true" />
                {t.openLecture}
              </Link>
            ) : (
              <Button loading={starting} onClick={() => onStart(item.conceptId)}>
                {t.startLecture}
              </Button>
            )}
          </div>
        ) : null
      }
    >
      {item && (
        <div className="flex flex-col gap-3 text-sm">
          <dl className="flex flex-col gap-3">
            {item.title && item.title !== item.conceptName && (
              <div>
                <dt className="text-xs font-medium text-fg-muted">{t.lectureTitle}</dt>
                <dd className="text-fg">{item.title}</dd>
              </div>
            )}
            <div>
              <dt className="text-xs font-medium text-fg-muted">{t.whyToday}</dt>
              <dd className="text-fg">{recommendationReason(item)}</dd>
            </div>
            {item.practicalSituation && (
              <div>
                <dt className="text-xs font-medium text-fg-muted">{t.practicalSituation}</dt>
                <dd className="text-fg">{item.practicalSituation}</dd>
              </div>
            )}
          </dl>
          <p className="text-xs text-fg-muted">{t.dismissHint}</p>
          <InlineError error={dismiss.error} />
        </div>
      )}
    </BottomSheet>
  );
}
