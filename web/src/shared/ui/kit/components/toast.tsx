
import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../lib/utils";

const toastVariants = cva(
  // D1: 토큰만 사용. 상태색은 아이콘에만, 상자는 중립(elevated + hairline + overlay 그림자)
  "relative flex w-full max-w-sm items-start gap-3 rounded-md border border-line bg-canvas-elevated p-3 text-fg shadow-overlay transition-opacity duration-base animate-fade-in motion-reduce:animate-none",
  {
    variants: {
      variant: { success: "", error: "", warning: "", info: "" }
    },
    defaultVariants: {
      variant: "info"
    }
  }
);

const toastIconVariants = cva("h-5 w-5 shrink-0", {
  variants: {
    variant: {
      success: "text-success",
      error: "text-danger",
      warning: "text-warning",
      info: "text-accent-text"
    }
  },
  defaultVariants: {
    variant: "info"
  }
});

export type ToastVariant = "success" | "error" | "warning" | "info";

interface ToastProps extends VariantProps<typeof toastVariants> {
  id: string;
  message: string;
  onDismiss: (id: string) => void;
  duration?: number;
}

export const Toast = React.forwardRef<HTMLDivElement, ToastProps>(
  ({ id, message, variant = "info", onDismiss, duration = 5000 }, ref) => {
    const [isExiting, setIsExiting] = React.useState(false);

    React.useEffect(() => {
      if (duration <= 0) return;

      const timer = setTimeout(() => {
        setIsExiting(true);
        setTimeout(() => onDismiss(id), 300);
      }, duration);

      return () => clearTimeout(timer);
    }, [id, duration, onDismiss]);

    function handleDismiss() {
      setIsExiting(true);
      setTimeout(() => onDismiss(id), 300);
    }

    const IconComponent = getIconForVariant(variant ?? "info");

    return (
      <div
        ref={ref}
        role={variant === "error" ? "alert" : "status"}
        aria-live={variant === "error" ? "assertive" : "polite"}
        aria-atomic="true"
        className={cn(toastVariants({ variant }), isExiting && "opacity-0")}
      >
        <IconComponent className={cn(toastIconVariants({ variant }))} />
        <div className="flex-1 py-0.5 text-sm font-medium">{message}</div>
        <button
          type="button"
          onClick={handleDismiss}
          className="relative -m-1 shrink-0 rounded-sm p-1 text-fg-muted transition-colors before:absolute before:-inset-2.5 before:content-[''] hover:bg-canvas-hover hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus"
          aria-label="알림 닫기"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M18 6 6 18" />
            <path d="m6 6 12 12" />
          </svg>
        </button>
      </div>
    );
  }
);
Toast.displayName = "Toast";

interface ToastContainerProps {
  toasts: Array<{
    id: string;
    message: string;
    variant: ToastVariant;
    duration?: number;
  }>;
  onDismiss: (id: string) => void;
}

export function ToastContainer({ toasts, onDismiss }: ToastContainerProps) {
  return (
    <div className="pointer-events-none fixed inset-x-3 bottom-[calc(var(--safe-bottom)+16px)] z-toast flex justify-center md:inset-x-auto md:bottom-auto md:right-4 md:top-4 md:block">
      <div className="pointer-events-auto flex flex-col gap-2">
        {toasts.map((toast) => (
          <Toast
            key={toast.id}
            id={toast.id}
            message={toast.message}
            variant={toast.variant}
            duration={toast.duration}
            onDismiss={onDismiss}
          />
        ))}
      </div>
    </div>
  );
}

interface ToastProviderProps {
  children: React.ReactNode;
}

export function ToastProvider({ children }: ToastProviderProps) {
  return <>{children}</>;
}

function getIconForVariant(variant: ToastVariant) {
  switch (variant) {
    case "success":
      return SuccessIcon;
    case "error":
      return ErrorIcon;
    case "warning":
      return WarningIcon;
    case "info":
    default:
      return InfoIcon;
  }
}

function SuccessIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <path d="m9 11 3 3L22 4" />
    </svg>
  );
}

function ErrorIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <circle cx="12" cy="12" r="10" />
      <path d="m15 9-6 6" />
      <path d="m9 9 6 6" />
    </svg>
  );
}

function WarningIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" />
      <path d="M12 9v4" />
      <path d="M12 17h.01" />
    </svg>
  );
}

function InfoIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <circle cx="12" cy="12" r="10" />
      <path d="M12 16v-4" />
      <path d="M12 8h.01" />
    </svg>
  );
}
