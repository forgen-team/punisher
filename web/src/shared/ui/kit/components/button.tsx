import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn, focusRing } from "../lib/utils";
import { Spinner } from "./spinner";

/** 시각 크기가 44px보다 작을 때 터치 영역만 44px로 넓힌다. */
const hitArea = "relative before:absolute before:inset-x-0 before:-inset-y-1.5 before:content-['']";

const buttonVariants = cva(
  cn(
    "inline-flex select-none items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium tracking-ui",
    "transition-colors duration-fast disabled:pointer-events-none disabled:opacity-50",
    focusRing
  ),
  {
    variants: {
      variant: {
        /** 화면당 1개 — 무채색 반전(강조색은 상태 표시에만 쓰므로 CTA에 쓰지 않는다) */
        primary: "bg-inverse text-fg-inverse shadow-xs hover:opacity-90",
        secondary: "border border-line bg-canvas-elevated text-fg shadow-xs hover:bg-canvas-hover",
        ghost: "text-fg-secondary hover:bg-canvas-hover hover:text-fg",
        danger: "bg-danger text-danger-on hover:opacity-90",
      },
      size: {
        sm: cn("h-8 px-3 text-sm", hitArea),
        md: "h-11 px-4 text-sm md:h-10",
      },
      block: { true: "w-full" },
    },
    defaultVariants: { variant: "primary", size: "md" },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  /** true면 Spinner 표시 + 비활성 + aria-busy */
  loading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className, variant, size, block, loading = false, disabled, type = "button", children, ...props },
  ref
) {
  return (
    <button
      ref={ref}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(buttonVariants({ variant, size, block }), className)}
      {...props}
    >
      {loading ? <Spinner size="sm" label={null} /> : null}
      {children}
    </button>
  );
});

const iconButtonVariants = cva(
  cn(
    "inline-flex shrink-0 items-center justify-center rounded-md transition-colors duration-fast",
    "disabled:pointer-events-none disabled:opacity-50 [&_svg]:h-5 [&_svg]:w-5",
    focusRing
  ),
  {
    variants: {
      variant: {
        ghost: "text-fg-secondary hover:bg-canvas-hover hover:text-fg",
        secondary: "border border-line bg-canvas-elevated text-fg shadow-xs hover:bg-canvas-hover",
      },
      size: {
        sm: cn(hitArea, "h-8 w-8 before:-inset-x-1.5"),
        md: "h-11 w-11 md:h-10 md:w-10",
      },
      selected: { true: "bg-canvas-active text-fg hover:bg-canvas-active hover:text-fg" },
    },
    defaultVariants: { variant: "ghost", size: "md" },
  }
);

export interface IconButtonProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "aria-label">,
    Omit<VariantProps<typeof iconButtonVariants>, "selected"> {
  /** 필수: 아이콘만 있으므로 스크린 리더용 이름 */
  "aria-label": string;
  /** 토글형 아이콘 버튼의 선택 상태 (aria-pressed 로 노출) */
  pressed?: boolean;
}

export const IconButton = React.forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { className, variant, size, pressed, type = "button", ...props },
  ref
) {
  return (
    <button
      ref={ref}
      type={type}
      aria-pressed={pressed}
      className={cn(iconButtonVariants({ variant, size, selected: pressed ?? false }), className)}
      {...props}
    />
  );
});

export { buttonVariants, iconButtonVariants };
