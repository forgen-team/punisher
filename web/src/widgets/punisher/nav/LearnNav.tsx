import { Settings } from "lucide-react";
import { isWitems } from "@/src/shared/config/runtime";
import { Link, usePathname, useRouter } from "@/src/shared/lib/router";
import { PageHeader, SegmentedTabs, cn, focusRing } from "@/src/shared/ui/kit";
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
  // Witems 셸은 화면 제목을 스스로 보이고 상단 바에 설정이 없으므로, 제목 대신 탭 옆에 학습 설정 링크를 둔다.
  const inWitems = isWitems();

  return (
    <div className="shrink-0">
      {/* 단독판 셸 상단 바는 화면 제목을 보이지 않으므로 md 이상에서 여기 둔다(witems 는 lg — docs/SYNC.md). */}
      {inWitems ? null : <PageHeader title={t.title} className="hidden pb-2 md:flex" />}
      <div className="flex items-center gap-2">
        <SegmentedTabs
          aria-label={t.tabsLabel}
          items={SEGMENTS.map((value) => ({ value, label: t[value] }))}
          value={segment}
          onChange={(next) => router.push(HREFS[next])}
          fill
          className="min-w-0 flex-1 md:max-w-sm md:flex-none"
        />
        {inWitems ? (
          <Link
            href="/settings"
            aria-label={t.settings}
            className={cn("inline-flex min-h-touch shrink-0 items-center gap-1.5 rounded-md px-2 text-sm text-fg-secondary hover:bg-canvas-hover hover:text-fg", focusRing)}
          >
            <Settings className="h-4 w-4" aria-hidden="true" />
            <span className="hidden sm:inline">{t.settings}</span>
          </Link>
        ) : null}
      </div>
    </div>
  );
}
