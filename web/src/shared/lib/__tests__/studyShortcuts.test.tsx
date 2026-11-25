import { act, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useStudyShortcuts, isTypingTarget } from "../shortcuts";
import { sanitize, setUiPrefs, DEFAULT_UI_PREFS } from "../ui-prefs";

function Keys({ onOne }: { onOne: () => void }) {
  useStudyShortcuts({ "1": onOne });
  return <textarea aria-label="answer" />;
}

afterEach(() => act(() => setUiPrefs(DEFAULT_UI_PREFS)));

describe("study shortcuts", () => {
  it("fires on the page, not while typing, and not when turned off", () => {
    const onOne = vi.fn();
    const { getByLabelText } = render(<Keys onOne={onOne} />);
    act(() => { window.dispatchEvent(new KeyboardEvent("keydown", { key: "1" })); });
    expect(onOne).toHaveBeenCalledTimes(1);
    getByLabelText("answer").dispatchEvent(new KeyboardEvent("keydown", { key: "1", bubbles: true }));
    expect(onOne).toHaveBeenCalledTimes(1);
    act(() => { window.dispatchEvent(new KeyboardEvent("keydown", { key: "1", metaKey: true })); });
    expect(onOne).toHaveBeenCalledTimes(1);
    act(() => setUiPrefs({ shortcuts: false }));
    act(() => { window.dispatchEvent(new KeyboardEvent("keydown", { key: "1" })); });
    expect(onOne).toHaveBeenCalledTimes(1);
  });

  it("knows typing targets", () => {
    expect(isTypingTarget(document.createElement("input"))).toBe(true);
    expect(isTypingTarget(document.createElement("button"))).toBe(false);
  });

  it("sanitizes stored prefs", () => {
    expect(sanitize({ fontSize: "huge" as never, shortcuts: "yes" as never, lineWidth: "wide" })).toEqual({ shortcuts: true, fontSize: "md", lineWidth: "wide" });
  });
});
