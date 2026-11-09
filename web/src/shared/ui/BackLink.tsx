import { ChevronLeft } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { cn, focusRing } from "@/src/shared/ui/kit";

/** 하위 화면 맨 위 '← 상위' 링크 — 단독판 셸 상단 바에는 뒤로 버튼이 없어 화면마다 같은 자리에 둔다. */
export function BackLink({ href, label }: { href: string; label: string }) {
  return (
    <Link
      href={href}
      className={cn(
        "-ml-2 inline-flex min-h-touch w-fit items-center gap-1 rounded-md px-2 text-sm text-fg-secondary hover:text-fg",
        focusRing,
      )}
    >
      <ChevronLeft className="h-4 w-4" aria-hidden="true" />
      {label}
    </Link>
  );
}
