import { Link } from "@/src/shared/lib/router";
import { PlayCircle } from "lucide-react";
import { Button, List, ListRow, Menu, cn } from "@/src/shared/ui/kit";
import { groupByParent, lectureHref } from "@/src/entities/punisher";
import type { Recommendation } from "@/src/shared/api/punisher";
import { DeeperGroup } from "@/src/shared/ui/punisher";
import { resumePosition } from "./resumeProgress";
import type { DismissedLecture } from "./useResumeDismiss";

/** 처음 보여 줄 묶음 수 — 더 있으면 '모두 보기'로 펼친다. */
const INITIAL_GROUPS = 5;

function resumeKey(item: Recommendation): string {
  return item.lectureId ?? item.conceptId;
}

interface ResumeRowProps {
  item: Recommendation;
  onDismiss: (target: DismissedLecture) => void;
  dismissing: boolean;
}

/** 이어보기 행: 개념 이름 / 강의 제목 / 읽던 부분 · 진행률 · 남은 시간. 누르면 저장된 위치로 이동. */
function ResumeRow({ item, onDismiss, dismissing }: ResumeRowProps) {
  if (!item.lectureId) { return null; }
  const percent = Math.min(100, Math.max(0, item.percent ?? 0));
  const title = item.conceptName || item.title;
  const subtitle = item.title && item.title !== item.conceptName ? item.title : "";
  const position = resumePosition(item.anchor);
  return (
    <ListRow
      href={`${lectureHref(item.lectureId)}${item.anchor ? `#pa-${item.anchor}` : ""}`}
      linkAs={Link}
      emphasized
      leading={<PlayCircle aria-hidden="true" className="text-accent-text" />}
      title={title}
      metaLines="all"
      meta={
        <>
          {subtitle && <span className="block whitespace-normal break-keep text-fg-secondary">{subtitle}</span>}
          {position && `${position} · `}{percent}% 읽음 · 약 {item.estimatedMinutes}분 남음
        </>
      }
      trailing={
        <Menu
          label={`${title} 강의 메뉴`}
          items={[{ key: "dismiss", label: "이어보기에서 제거", disabled: dismissing, onSelect: () => onDismiss({ lectureId: item.lectureId as string, label: title }) }]}
        />
      }
    />
  );
}

interface ResumeListProps {
  /** 최근에 본 순서의 이어보기 항목 */
  items: Recommendation[];
  /** 폭이 넓으면 항목을 2~3열로 펼친다(전체 폭 영역용) */
  columns?: boolean;
  dismissing: boolean;
  onDismiss: (target: DismissedLecture) => void;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
}

/** 읽다 만 강의 전부. "더 깊게" 본 강의는 원 강의 아래로 묶고, 많으면 앞의 5묶음만 먼저 보여 준다. */
export function ResumeList({ items, columns = false, dismissing, onDismiss, expanded, onExpandedChange }: ResumeListProps) {
  const groups = groupByParent(items, resumeKey, (it) => it.parentLectureId);
  const visible = expanded ? groups : groups.slice(0, INITIAL_GROUPS);
  const hidden = groups.slice(INITIAL_GROUPS).reduce((sum, group) => sum + 1 + group.children.length, 0);
  const count = visible.length;

  function renderRow(item: Recommendation) {
    return <ResumeRow key={resumeKey(item)} item={item} dismissing={dismissing} onDismiss={onDismiss} />;
  }

  return (
    <>
      <ul role="list" className={cn("grid grid-cols-1", columns && count >= 2 && "fold:grid-cols-2 fold:gap-x-6", columns && count >= 3 && "xl:grid-cols-3")}>
        {visible.map(({ item, children }, index) => (
          <li
            key={resumeKey(item)}
            className={cn(
              "min-w-0 border-t border-line",
              index === 0 && "border-t-0",
              columns && index === 1 && "fold:border-t-0",
              columns && index === 2 && count >= 3 && "xl:border-t-0",
            )}
          >
            <List>
              <DeeperGroup parent={renderRow(item)}>{children.map(renderRow)}</DeeperGroup>
            </List>
          </li>
        ))}
      </ul>
      {hidden > 0 && (
        <div className="mt-2 flex justify-center">
          <Button variant="ghost" size="sm" onClick={() => onExpandedChange(!expanded)}>
            {expanded ? "접기" : `모두 보기 (${hidden}개 더)`}
          </Button>
        </div>
      )}
    </>
  );
}
