import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
import "./styles/globals.css";

const root = document.getElementById("root");
if (!root) { throw new Error("#root not found"); }

// 오프라인 읽기(단독판만 — Witems 앱 모드는 mount.tsx 로 들어와 여기를 지나지 않는다).
if (import.meta.env.PROD && "serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch((err: unknown) => console.warn("punisher: service worker not registered", err));
  });
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
