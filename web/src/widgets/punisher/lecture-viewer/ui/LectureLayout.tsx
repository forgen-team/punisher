
import type { ReactNode } from "react";
import type { LectureSection } from "@/src/shared/api/punisher";
import { SectionToc } from "./SectionToc";
import { TocRail } from "./TocRail";

const STICKY_TOP = "sticky top-[calc(var(--app-header-offset)+4rem)] shrink-0 self-start pt-4";
// 펼친 열은 화면 높이 안에서 따로 스크롤. 접은 목차는 펼침 패널이 열 밖으로 나가야 해서 overflow 를 두지 않는다.
const STICKY = `${STICKY_TOP} max-h-[calc(100dvh-var(--app-header-offset)-5rem)] overflow-y-auto pb-10`;

interface LectureLayoutProps {
  sections: LectureSection[];
  totalMinutes: number;
  activeAnchor: string;
  /** 오른쪽 보조 열(xl 이상). 없으면 열을 두지 않는다 */
  assist?: ReactNode;
  children: ReactNode;
}

/**
 * 강의 읽기 배치 — 본문은 가독 폭(max-w-2xl)을 지키고 남는 폭은 양옆 열로 쓴다.
 * - lg 미만: 본문만(목차·질문/메모는 상단 바 시트)
 * - lg~xl: 접은 목차(아이콘 + 섹션 점) | 본문
 * - xl~2xl: 접은 목차 | 본문 | 보조 열(w-72)
 * - 2xl 이상: 펼친 목차(w-56) | 본문 | 보조 열(w-80)
 * 목차가 없을 때(생성 시작 직후)도 열 자리는 그대로 둬서 섹션이 생길 때 본문이 옆으로 밀리지 않게 한다.
 */
export function LectureLayout({ sections, totalMinutes, activeAnchor, assist, children }: LectureLayoutProps) {
  const hasToc = sections.length > 0;
  return (
    <div className="flex gap-6 2xl:gap-8">
      <aside aria-label="목차" className={`${STICKY_TOP} z-panel hidden w-10 lg:block 2xl:hidden`}>
        {hasToc && <TocRail sections={sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} />}
      </aside>
      <aside aria-label="목차" className={`${STICKY} hidden w-56 2xl:block`}>
        {hasToc && <SectionToc sections={sections} totalMinutes={totalMinutes} activeAnchor={activeAnchor} />}
      </aside>
      <article className="mx-auto w-full min-w-0 max-w-2xl pb-16">{children}</article>
      {assist && <aside className={`${STICKY} hidden w-72 xl:block 2xl:w-80`}>{assist}</aside>}
    </div>
  );
}
