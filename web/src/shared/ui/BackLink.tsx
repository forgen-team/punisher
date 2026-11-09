import { ChevronLeft } from "lucide-react";
import { Link, useRouter } from "@/src/shared/lib/router";
import { canGoBack } from "@/src/shared/lib/in-app-history";
import { cn, focusRing } from "@/src/shared/ui/kit";

/**
 * 하위 화면 맨 위 '← 상위' — 앱 안에서 들어왔으면 들어온 곳으로(브라우저 뒤로), 직접 들어왔으면 href.
 * witems DesktopBackLink 와 같은 판단. 단독판 셸 상단 바에는 ← 가 없어 모든 폭에서 보인다.
 */
export function BackLink({ href, label, className }: { href: string; label: string; className?: string }) {
  const router = useRouter();
  return (
    <Link
      href={href}
      onClick={(event) => {
        if (canGoBack()) {
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
