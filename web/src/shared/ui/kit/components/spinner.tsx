import * as React from "react";
import { cn } from "../lib/utils";
import { useT } from "@/src/shared/i18n";
import { kitMessages } from "../lib/messages";

export interface SpinnerProps extends React.HTMLAttributes<HTMLSpanElement> {
  size?: "sm" | "md";
  /** 스크린 리더용 문구. 버튼 안처럼 장식용이면 null */
  label?: string | null;
}

/** 원형 로딩 표시. currentColor를 따르므로 부모 글자색에 맞춰진다. */
export function Spinner({ size = "md", label, className, ...props }: SpinnerProps) {
  const t = useT(kitMessages);
  const text = label === undefined ? t.loading : label;
  return (
    <span
      role={text ? "status" : undefined}
      aria-hidden={text ? undefined : true}
      className={cn("inline-flex shrink-0 items-center justify-center", className)}
      {...props}
    >
      <span
        className={cn(
          "animate-spin rounded-full border-2 border-current border-r-transparent opacity-70 motion-reduce:animate-none",
          size === "sm" ? "h-4 w-4" : "h-5 w-5"
        )}
      />
      {text ? <span className="sr-only">{text}</span> : null}
    </span>
  );
}
