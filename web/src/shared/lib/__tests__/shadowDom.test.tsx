import { useRef } from "react";
import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setWitemsRuntime } from "@/src/shared/config/runtime";
import { FeedbackHost, Menu, confirmAction, resetFeedback, useFocusTrap } from "@/src/shared/ui/kit";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { activeAppElement, appRoot, eventWithin, getAppElementById, queryAppAll } from "../dom-root";

/**
 * Witems 는 앱을 Shadow DOM 안에 그린다(계약 §3·§4). 운영에서 본 결함 — 본문 앵커를 document 로 찾음(1),
 * ⋯ 메뉴가 클릭 전에 닫힘(2), 포커스 판정(13) — 이 Shadow DOM 안에서 맞게 동작하는지 jsdom attachShadow 로 확인한다.
 */
let host: HTMLDivElement;
let shadow: ShadowRoot;
let el: HTMLDivElement;
let portal: HTMLDivElement;

beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  shadow = host.attachShadow({ mode: "open" });
  el = document.createElement("div");
  portal = document.createElement("div");
  shadow.append(el, portal);
  setWitemsRuntime({ apiBase: "/apps-api/punisher/", portal, setTitle: () => undefined });
});

afterEach(() => {
  act(() => resetFeedback());
  setWitemsRuntime(null);
  host.remove();
});

function pointerDown(target: Element) {
  // 실제 브라우저처럼 Shadow 경계를 넘는(composed) 이벤트 — document 에서 보면 target 은 호스트다.
  target.dispatchEvent(new Event("pointerdown", { bubbles: true, composed: true }));
}

describe("dom-root (Shadow DOM)", () => {
  it("finds anchors inside the shadow root, not in document", () => {
    el.innerHTML = `<section id="pa-intro" data-lecture-anchor="intro"></section><section id="pa-tcp" data-lecture-anchor="tcp"></section>`;
    expect(document.getElementById("pa-intro")).toBeNull();
    expect(appRoot()).toBe(shadow);
    expect(getAppElementById("pa-intro")).toBe(el.firstElementChild);
    expect(queryAppAll("[data-lecture-anchor]")).toHaveLength(2);
  });

  it("scrolls to a lecture anchor drawn in the shadow root", () => {
    el.innerHTML = `<section id="pa-tcp" data-lecture-anchor="tcp"></section>`;
    const spy = vi.fn();
    Element.prototype.scrollIntoView = spy;
    expect(scrollToAnchor("tcp", "auto")).toBe(true);
    expect(spy).toHaveBeenCalledTimes(1);
    expect(scrollToAnchor("missing", "auto")).toBe(false);
  });

  it("tells inside from outside with composedPath although document sees the host", () => {
    el.innerHTML = `<div id="panel"><button id="inner">x</button></div><button id="other">y</button>`;
    const panel = getAppElementById("panel");
    const seen: Array<{ target: EventTarget | null; inside: boolean }> = [];
    const listener = (event: Event) => seen.push({ target: event.target, inside: eventWithin(event, panel) });
    document.addEventListener("pointerdown", listener);
    pointerDown(getAppElementById("inner") as HTMLElement);
    pointerDown(getAppElementById("other") as HTMLElement);
    document.removeEventListener("pointerdown", listener);
    expect(seen[0]).toEqual({ target: host, inside: true });
    expect(seen[1]).toEqual({ target: host, inside: false });
  });

  it("reads the focused element inside the shadow root", () => {
    el.innerHTML = `<button id="b">b</button>`;
    (getAppElementById("b") as HTMLElement).focus();
    expect(document.activeElement).toBe(host);
    expect(activeAppElement()).toBe(getAppElementById("b"));
  });
});

describe("Menu inside the shadow root", () => {
  it("keeps the menu open on a pointerdown on its item, so the click selects it", () => {
    const onSelect = vi.fn();
    const { getByRole, queryByRole } = render(<Menu items={[{ key: "a", label: "지도에서 보기", onSelect }]} label="강의 메뉴" />, { container: el });
    fireEvent.click(getByRole("button", { name: "강의 메뉴" }));
    const item = getByRole("menuitem", { name: "지도에서 보기" });
    act(() => pointerDown(item));
    expect(queryByRole("menu")).not.toBeNull();
    fireEvent.click(item);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("closes on a pointerdown outside the menu", () => {
    // React 가 그리는 컨테이너(el) 밖, 같은 Shadow DOM 안의 요소
    const outside = document.createElement("p");
    shadow.append(outside);
    const { getByRole, queryByRole } = render(<Menu items={[{ key: "a", label: "A", onSelect: () => undefined }]} label="메뉴" />, { container: el });
    fireEvent.click(getByRole("button", { name: "메뉴" }));
    act(() => pointerDown(outside));
    expect(queryByRole("menu")).toBeNull();
  });
});

function Trap({ active }: { active: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  useFocusTrap(ref, active);
  return (
    <div ref={ref}>
      <button type="button">first</button>
      <button type="button" data-autofocus="">second</button>
    </div>
  );
}

describe("useFocusTrap inside the shadow root", () => {
  it("focuses inside, keeps Tab inside and gives focus back on close", () => {
    const opener = document.createElement("button");
    el.append(opener);
    opener.focus();
    const { getByText, rerender } = render(<Trap active />, { container: el.appendChild(document.createElement("div")) });
    expect(shadow.activeElement).toBe(getByText("second"));
    fireEvent.keyDown(getByText("second"), { key: "Tab" });
    expect(el.contains(shadow.activeElement)).toBe(true);
    rerender(<Trap active={false} />);
    expect(shadow.activeElement).toBe(opener);
  });
});

describe("confirmAction inside the shadow root", () => {
  it("opens in ctx.portal with Cancel focused and resolves with the choice", async () => {
    render(<FeedbackHost />, { container: el });
    let answer: Promise<boolean> = Promise.resolve(false);
    act(() => {
      answer = confirmAction({ title: "목표를 삭제할까요?", confirmLabel: "삭제", danger: true });
    });
    const dialog = portal.querySelector("[role='alertdialog']");
    expect(dialog).not.toBeNull();
    expect(document.querySelector("[role='alertdialog']")).toBeNull();
    expect(shadow.activeElement?.textContent).toBe("취소");
    const confirm = Array.from(portal.querySelectorAll("button")).find((b) => b.textContent === "삭제") as HTMLButtonElement;
    act(() => confirm.click());
    await expect(answer).resolves.toBe(true);
    expect(portal.querySelector("[role='alertdialog']")).toBeNull();
  });
});
