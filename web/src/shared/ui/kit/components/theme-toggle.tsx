
import * as React from "react";
import { useAtom, useAtomValue, useSetAtom } from "jotai";
import {
  themePreferenceAtom,
  resolvedThemeAtom,
  toggleThemeAtom,
  cycleThemeAtom,
  type ThemePreference
} from "@/src/shared/lib/theme";
import { cn } from "../lib/utils";

// Icons for theme states
function SunIcon({ className }: { className?: string }) {
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
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2" />
      <path d="M12 20v2" />
      <path d="m4.93 4.93 1.41 1.41" />
      <path d="m17.66 17.66 1.41 1.41" />
      <path d="M2 12h2" />
      <path d="M20 12h2" />
      <path d="m6.34 17.66-1.41 1.41" />
      <path d="m19.07 4.93-1.41 1.41" />
    </svg>
  );
}

function MoonIcon({ className }: { className?: string }) {
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
      <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
    </svg>
  );
}

function MonitorIcon({ className }: { className?: string }) {
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
      <rect width="20" height="14" x="2" y="3" rx="2" />
      <line x1="8" x2="16" y1="21" y2="21" />
      <line x1="12" x2="12" y1="17" y2="21" />
    </svg>
  );
}

interface ThemeToggleProps {
  /** Show tooltip with current theme name */
  showTooltip?: boolean;
  /** Button size */
  size?: "sm" | "md" | "lg";
  /** Additional class names */
  className?: string;
  /** Toggle mode: 'toggle' switches between light/dark, 'cycle' includes system option */
  mode?: "toggle" | "cycle";
}

export function ThemeToggle({
  showTooltip = true,
  size = "md",
  className,
  mode = "toggle"
}: ThemeToggleProps) {
  const preference = useAtomValue(themePreferenceAtom);
  const resolvedTheme = useAtomValue(resolvedThemeAtom);
  const toggle = useSetAtom(toggleThemeAtom);
  const cycle = useSetAtom(cycleThemeAtom);

  const handleClick = () => {
    if (mode === "cycle") {
      cycle();
    } else {
      toggle();
    }
  };

  const sizeClasses = {
    sm: "h-8 w-8",
    md: "h-9 w-9",
    lg: "h-10 w-10"
  };

  const iconSizeClasses = {
    sm: "h-4 w-4",
    md: "h-5 w-5",
    lg: "h-6 w-6"
  };

  const getIcon = () => {
    if (mode === "cycle") {
      switch (preference) {
        case "light":
          return <SunIcon className={iconSizeClasses[size]} />;
        case "dark":
          return <MoonIcon className={iconSizeClasses[size]} />;
        case "system":
          return <MonitorIcon className={iconSizeClasses[size]} />;
      }
    }
    return resolvedTheme === "dark" ? (
      <MoonIcon className={iconSizeClasses[size]} />
    ) : (
      <SunIcon className={iconSizeClasses[size]} />
    );
  };

  const getLabel = () => {
    if (mode === "cycle") {
      switch (preference) {
        case "light":
          return "Light mode";
        case "dark":
          return "Dark mode";
        case "system":
          return "System theme";
      }
    }
    return resolvedTheme === "dark" ? "Switch to light mode" : "Switch to dark mode";
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "relative inline-flex items-center justify-center rounded-lg",
        "bg-transparent hover:bg-canvas-hover",
        "text-fg-muted hover:text-fg",
        "transition-all duration-200",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus",
        sizeClasses[size],
        className
      )}
      title={showTooltip ? getLabel() : undefined}
      aria-label={getLabel()}
    >
      <span className="sr-only">{getLabel()}</span>
      {getIcon()}
    </button>
  );
}

interface ThemeSelectProps {
  className?: string;
}

export function ThemeSelect({ className }: ThemeSelectProps) {
  const [preference, setPreference] = useAtom(themePreferenceAtom);

  const options: { value: ThemePreference; label: string; icon: React.ReactNode }[] = [
    { value: "light", label: "라이트", icon: <SunIcon className="h-4 w-4" /> },
    { value: "dark", label: "다크", icon: <MoonIcon className="h-4 w-4" /> },
    { value: "system", label: "시스템", icon: <MonitorIcon className="h-4 w-4" /> }
  ];

  return (
    <div className={cn("flex items-center gap-1 p-1 rounded-lg bg-canvas-subtle", className)}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          onClick={() => setPreference(option.value)}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-all",
            preference === option.value
              ? "bg-canvas-elevated text-fg shadow-sm"
              : "text-fg-muted hover:text-fg"
          )}
        >
          {option.icon}
          <span>{option.label}</span>
        </button>
      ))}
    </div>
  );
}

export { SunIcon, MoonIcon, MonitorIcon };
