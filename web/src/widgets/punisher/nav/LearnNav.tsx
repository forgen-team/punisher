import { usePathname, useRouter } from "@/src/shared/lib/router";
import { PageHeader, SegmentedTabs } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

type LearnSegment = "lectures" | "map" | "history";

const HREFS: Record<LearnSegment, string> = {
  lectures: "/",
  map: "/map",
  history: "/history",
};
const SEGMENTS = ["lectures", "map", "history"] as const;

function segmentOf(pathname: string): LearnSegment | null {
  if (pathname === "/" || pathname === "") { return "lectures"; }
  if (pathname.startsWith("/map") || pathname.startsWith("/goals")) { return "map"; }
  if (pathname.startsWith("/history")) { return "history"; }
  return null;
}

/**
 * 학습 화면 상단: 제목(데스크톱만) + 세그먼트 [강의·지도·기록].
 * 강의 뷰어·즉석 강의·설정 화면에서는 자체 상단 바를 쓰므로 렌더하지 않는다.
 */
export function LearnNav() {
  const pathname = usePathname();
  const router = useRouter();
  const segment = segmentOf(pathname);
  const t = useT(messages);
  if (!segment) { return null; }

  return (
    <div className="shrink-0">
      {/* 단독판 셸 상단 바는 화면 제목을 보이지 않으므로 md 이상에서 여기 둔다(witems 는 lg — docs/SYNC.md). */}
      <PageHeader title={t.title} className="hidden pb-2 md:flex" />
      <SegmentedTabs
        aria-label={t.tabsLabel}
        items={SEGMENTS.map((value) => ({ value, label: t[value] }))}
        value={segment}
        onChange={(next) => router.push(HREFS[next])}
        fill
        className="md:max-w-sm"
      />
    </div>
  );
}
