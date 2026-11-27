import { PlayCircle } from "lucide-react";
import { ListRow, Menu } from "@forgen-team/ui";
import { isReadThrough, lectureAnchorHref, readThroughLabel } from "@/src/entities/punisher";
import type { Recommendation } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

export function resumeKey(item: Recommendation): string {
  return item.lectureId ?? item.conceptId;
}

interface ResumeRowProps {
  item: Recommendation;
  dismissing: boolean;
  onDismiss: (lectureId: string) => void;
}

/** 이어보기 행: 개념 이름 / "강의 제목 · n% 읽음". 누르면 읽던 자리(`?at=`), ⋯ "이어보기에서 빼기". List 안에서 쓴다. */
export function ResumeRow({ item, dismissing, onDismiss }: ResumeRowProps) {
  const locale = useLocale();
  const t = useT(messages);
  const lectureId = item.lectureId;
  if (!lectureId) { return null; }
  const percent = Math.min(100, Math.max(0, item.percent ?? 0));
  const title = item.conceptName || item.title;
  const subtitle = item.title && item.title !== item.conceptName ? item.title : "";
  const progress = isReadThrough(percent, false) ? readThroughLabel(locale) : t.percentRead(percent);
  return (
    <ListRow
      href={lectureAnchorHref(lectureId, item.anchor)}
      leading={<PlayCircle aria-hidden="true" className="text-accent-text" />}
      title={title}
      meta={subtitle ? `${subtitle} · ${progress}` : progress}
      trailing={
        <Menu
          label={t.menu(title)}
          items={[{ key: "dismiss", label: t.dismiss, disabled: dismissing, onSelect: () => onDismiss(lectureId) }]}
        />
      }
    />
  );
}
