import { ChevronLeft } from "lucide-react";
import { Link, useRouter } from "@/src/shared/lib/router";
import { canGoBack } from "@/src/shared/lib/in-app-history";
import { cn, focusRing } from "@/src/shared/lib/cn";
import { isWitems } from "@/src/shared/config/runtime";

/**
 * 하위 화면 맨 위 '← 상위' — 앱 안에서 들어왔으면 들어온 곳으로(브라우저 뒤로), 직접 들어왔으면 href.
 * witems DesktopBackLink 와 같은 판단. 단독판 셸 상단 바에는 ← 가 없어 모든 폭에서 보인다.
 * `replace`: 들어온 곳과 관계없이 href 로(기록을 쌓지 않고) — 한 단 상세에서 목록으로 돌아갈 때(witems 상단 바 toList).
 * Witems 앱 모드에서는 셸이 ← 를 그리므로(UI-PAGE-LAYOUT §1) 그리지 않는다.
 */
export function BackLink({ href, label, replace = false, className }: { href: string; label: string; replace?: boolean; className?: string }) {
  const router = useRouter();
  if (isWitems()) { return null; }
  return (
    <Link
      href={href}
      replace={replace}
      onClick={(event) => {
        if (!replace && canGoBack()) {
          event.preventDefault();
          router.back();
        }
      }}
      className={cn(
        "-ml-2 inline-flex min-h-touch w-fit items-center gap-1 rounded-md px-2 text-sm text-fg-secondary hover:text-fg",
        focusRing,
        className,
      )}
    >
      <ChevronLeft className="h-4 w-4" aria-hidden="true" />
      {label}
    </Link>
  );
}
