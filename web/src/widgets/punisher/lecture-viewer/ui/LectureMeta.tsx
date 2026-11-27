import { areaLabel } from "@/src/entities/punisher";
import type { LinkedLecture } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { ParentLectureLink } from "./DeeperLectures";

interface LectureMetaProps {
  area?: string;
  minutes: number;
  /** 심화(또는 다시 만든) 강의면 원 강의 */
  parent?: LinkedLecture;
}

/** 메타 한 줄: 영역 · 약 n분 · "○○ 강의에서 이어짐"(심화일 때). */
export function LectureMeta({ area, minutes, parent }: LectureMetaProps) {
  const t = useT(messages);
  const parts = [area ? areaLabel(area) : "", minutes ? t.aboutMinutes(minutes) : ""].filter(Boolean);
  return (
    <span className="text-sm">
      {parts.join(" · ")}
      {parent && (
        <>
          {parts.length > 0 && " · "}
          <ParentLectureLink parent={parent} />
        </>
      )}
    </span>
  );
}
