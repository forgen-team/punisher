import * as React from "react";
import { cn, focusRing } from "../lib/utils";

/**
 * 입력 계열 공통 스타일. 모바일 16px(iOS 확대 방지) → md 이상 14px.
 * 경계선은 border-line-strong(대비 3:1 이상)을 써서 입력 영역이 식별되게 한다.
 */
export const controlBase = cn(
  "w-full rounded-md border border-line-strong bg-canvas-elevated px-3 text-base text-fg md:text-sm",
  "placeholder:text-fg-muted transition-colors duration-fast",
  "disabled:cursor-not-allowed disabled:bg-canvas-subtle disabled:opacity-60",
  "aria-[invalid=true]:border-danger",
  focusRing,
  "focus-visible:ring-offset-0 focus-visible:border-focus"
);

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

export const Input = React.forwardRef<HTMLInputElement, InputProps>(function Input({ className, ...props }, ref) {
  return <input ref={ref} className={cn(controlBase, "h-11 md:h-10", className)} {...props} />;
});

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, rows = 3, ...props },
  ref
) {
  return <textarea ref={ref} rows={rows} className={cn(controlBase, "min-h-20 py-2", className)} {...props} />;
});
