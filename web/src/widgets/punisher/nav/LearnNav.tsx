import { usePathname, useRouter } from "@/src/shared/lib/router";
import { PageHeader, SegmentedTabs } from "@/src/shared/ui/kit";

type LearnSegment = "lectures" | "map" | "history";

const HREFS: Record<LearnSegment, string> = {
  lectures: "/",
  map: "/map",
  history: "/history",
};
const ITEMS = [
  { value: "lectures", label: "강의" },
  { value: "map", label: "지도" },
  { value: "history", label: "기록" },
] as const;

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
  if (!segment) { return null; }

  return (
    <div className="shrink-0">
      <PageHeader title="학습" className="hidden pb-2 md:flex" />
      <SegmentedTabs
        aria-label="학습 보기"
        items={ITEMS}
        value={segment}
        onChange={(next) => router.push(HREFS[next])}
        fill
        className="md:max-w-sm"
      />
    </div>
  );
}
