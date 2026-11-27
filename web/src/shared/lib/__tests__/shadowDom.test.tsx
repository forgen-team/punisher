import { useRef } from "react";
import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { setWitemsRuntime } from "@/src/shared/config/runtime";
import { FeedbackProvider, UIProvider, useConfirm } from "@forgen-team/ui";
import { useFocusTrap } from "../useFocusTrap";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { activeAppElement, appRoot, eventWithin, getAppElementById, queryAppAll } from "../dom-root";

/**
 * Witems 는 앱을 Shadow DOM 안에 그린다(계약 §3·§4). 운영에서 본 결함 — 본문 앵커를 document 로 찾음(1),
 * 포커스 판정(13) — 이 Shadow DOM 안에서 맞게 동작하는지 jsdom attachShadow 로 확인한다.
 * ⋯ 메뉴(2)는 @forgen-team/ui Menu(SEED)로 바뀌어 그 패키지 테스트가 맡는다.
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

let ask: ReturnType<typeof useConfirm> | null = null;
function ConfirmProbe() {
  ask = useConfirm();
  return null;
}

describe("useConfirm inside the shadow root", () => {
  it("opens the dialog inside the shadow root (not document) and resolves with the choice", async () => {
    // 앱 뿌리(app/UIRoot.tsx)처럼 portalRoot = ctx.portal. 확인 창은 FeedbackProvider 자리(앱 루트 = Shadow DOM 안)에 그려진다.
    render(
      <UIProvider portalRoot={portal}>
        <FeedbackProvider><ConfirmProbe /></FeedbackProvider>
      </UIProvider>,
      { container: el },
    );
    let answer: Promise<boolean> = Promise.resolve(false);
    act(() => {
      answer = (ask as NonNullable<typeof ask>)({ title: "목표를 삭제할까요?", confirmLabel: "삭제", tone: "danger" });
    });
    const dialog = shadow.querySelector("[role='alertdialog']");
    expect(dialog).not.toBeNull();
    expect(document.querySelector("[role='alertdialog']")).toBeNull();
    const confirm = Array.from(shadow.querySelectorAll("button")).find((b) => b.textContent === "삭제") as HTMLButtonElement;
    act(() => confirm.click());
    await expect(answer).resolves.toBe(true);
  });
});
