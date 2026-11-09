
import { useEffect, useState } from "react";

function readTheme(): "light" | "dark" {
  if (typeof document === "undefined") { return "light"; }
  return document.documentElement.getAttribute("data-theme") === "dark" ? "dark" : "light";
}

/** Tracks `<html data-theme>` (set by @witems/config ThemeProvider). */
export function useResolvedTheme(): "light" | "dark" {
  const [theme, setTheme] = useState<"light" | "dark">("light");
  useEffect(() => {
    setTheme(readTheme());
    const observer = new MutationObserver(() => setTheme(readTheme()));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => observer.disconnect();
  }, []);
  return theme;
}
