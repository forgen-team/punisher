
import * as React from "react";
import { cn, focusRing } from "../lib/utils";
import { controlBase } from "./input";

interface FieldContextValue {
  id: string;
  describedBy?: string;
  invalid: boolean;
}

const FieldContext = React.createContext<FieldContextValue | null>(null);

/** Field 안의 컨트롤이 id/aria-describedby/aria-invalid 를 자동으로 받도록 한다. */
export function useField() {
  return React.useContext(FieldContext);
}

export interface FieldProps {
  label: React.ReactNode;
  /** 보조 설명 (label 아래가 아니라 컨트롤 아래 표시) */
  hint?: React.ReactNode;
  /** 오류 문구 — 있으면 aria-invalid + role=alert */
  error?: React.ReactNode;
  required?: boolean;
  /** 직접 지정하지 않으면 useId 로 생성 */
  id?: string;
  className?: string;
  /** 컨트롤 1개. Input/Textarea/Select/Switch는 Field 컨텍스트를 자동으로 읽는다 */
  children: React.ReactNode;
}

/** Field — 레이블 + 컨트롤 + 힌트/오류. 레이블-컨트롤 연결과 오류 낭독을 책임진다. */
export function Field({ label, hint, error, required, id, className, children }: FieldProps) {
  const autoId = React.useId();
  const controlId = id ?? autoId;
  const hintId = hint ? `${controlId}-hint` : undefined;
  const errorId = error ? `${controlId}-error` : undefined;
  const describedBy = [errorId, hintId].filter(Boolean).join(" ") || undefined;
  const value = React.useMemo(
    () => ({ id: controlId, describedBy, invalid: Boolean(error) }),
    [controlId, describedBy, error]
  );

  return (
    <FieldContext.Provider value={value}>
      <div className={cn("space-y-1.5", className)}>
        <label htmlFor={controlId} className="block text-sm font-medium text-fg">
          {label}
          {required ? (
            <span className="ml-0.5 text-danger" aria-hidden="true">*</span>
          ) : null}
        </label>
        {children}
        {error ? (
          <p id={errorId} role="alert" className="break-keep text-xs text-danger">{error}</p>
        ) : hint ? (
          <p id={hintId} className="break-keep text-xs text-fg-muted">{hint}</p>
        ) : null}
      </div>
    </FieldContext.Provider>
  );
}

/** Field 컨텍스트 → 컨트롤 접근성 속성 (명시 props가 우선) */
function fieldProps<P extends { id?: string; "aria-describedby"?: string; "aria-invalid"?: React.AriaAttributes["aria-invalid"] }>(
  field: FieldContextValue | null,
  props: P
) {
  if (!field) return props;
  return {
    ...props,
    id: props.id ?? field.id,
    "aria-describedby": props["aria-describedby"] ?? field.describedBy,
    "aria-invalid": props["aria-invalid"] ?? (field.invalid || undefined),
  };
}

export type FieldInputProps = React.InputHTMLAttributes<HTMLInputElement>;

/** Field 안에서 쓰는 Input (컨텍스트 자동 연결). Field 밖에서는 Input 을 직접 써도 된다. */
export const FieldInput = React.forwardRef<HTMLInputElement, FieldInputProps>(function FieldInput(
  { className, ...props },
  ref
) {
  const field = useField();
  return <input ref={ref} className={cn(controlBase, "h-11 md:h-10", className)} {...fieldProps(field, props)} />;
});

export type FieldTextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export const FieldTextarea = React.forwardRef<HTMLTextAreaElement, FieldTextareaProps>(function FieldTextarea(
  { className, rows = 3, ...props },
  ref
) {
  const field = useField();
  return (
    <textarea ref={ref} rows={rows} className={cn(controlBase, "min-h-20 py-2", className)} {...fieldProps(field, props)} />
  );
});

export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement>;

/** Select — 네이티브 <select>. 모바일에서 OS 피커를 쓰므로 접근성·터치가 가장 좋다. */
export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className, children, ...props },
  ref
) {
  const field = useField();
  return (
    <div className="relative">
      <select
        ref={ref}
        className={cn(controlBase, "h-11 min-w-0 appearance-none truncate pr-9 md:h-10", className)}
        {...fieldProps(field, props)}
      >
        {children}
      </select>
      <svg
        viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true"
        className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-muted"
      >
        <path d="m6 9 6 6 6-6" />
      </svg>
    </div>
  );
});

export interface SwitchProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "onChange"> {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  /** Field 밖에서 쓸 때 필수 */
  "aria-label"?: string;
}

/** Switch — role="switch". 켜짐 상태만 강조색. 44px 터치 영역. */
export const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(function Switch(
  { checked, onCheckedChange, className, disabled, ...props },
  ref
) {
  const field = useField();
  const a11y = fieldProps(field, { id: props.id, "aria-describedby": props["aria-describedby"] });
  return (
    <button
      ref={ref}
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      {...props}
      id={a11y.id}
      aria-describedby={a11y["aria-describedby"]}
      className={cn(
        "relative inline-flex h-6 w-10 shrink-0 items-center rounded-full border transition-colors duration-fast",
        "before:absolute before:-inset-x-1 before:-inset-y-2.5 before:content-['']",
        "disabled:cursor-not-allowed disabled:opacity-50",
        checked ? "border-transparent bg-accent-solid" : "border-line-strong bg-canvas-hover",
        focusRing,
        className
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          "inline-block h-4 w-4 rounded-full shadow-sm transition-transform duration-fast",
          checked ? "translate-x-5 bg-accent-on" : "translate-x-1 bg-fg-muted"
        )}
      />
    </button>
  );
});
