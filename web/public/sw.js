/*
 * Punisher 오프라인 읽기(W5a, 단독판만 — Witems 앱 모드는 등록하지 않는다: docs/WITEMS-APP.md).
 * 읽기 전용: 한 번 연 강의·질문 해설·확인 화면과 오늘·카드 목록을 네트워크 우선으로 받아 두었다가, 맥이 잠들었거나
 * 연결이 끊기면 저장본을 보인다. 쓰기(채점·저장)는 캐시하지 않는다. Cache API 는 만료가 없어 개수로 정리한다.
 */
const VERSION = "punisher-offline-v1";
const SHELL = `${VERSION}-shell`;
const DATA = `${VERSION}-data`;
const MAX_DATA = 120;
const NETWORK_TIMEOUT_MS = 4000;

// 저장해 둘 읽기 API(GET). 생성 스트림(SSE)·내보내기는 뺀다.
const READABLE = [
  /^\/v1\/lectures\/[^/]+$/,
  /^\/v1\/lectures\/[^/]+\/(links|exercises|questions)$/,
  /^\/v1\/inquiries\/[^/]+$/,
  /^\/v1\/checks\/[^/]+$/,
  /^\/v1\/concepts\/[^/]+$/,
  /^\/v1\/today$/,
  /^\/v1\/cards$/,
  /^\/v1\/me\/(settings|activity|study-prefs|lectures|questions)$/,
  /^\/v1\/reviews\/(summary|forecast)$/,
  /^\/v1\/(status|notes|goals|graph|mastery)$/,
];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(SHELL).then((cache) => cache.add("/")).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => !k.startsWith(VERSION)).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

function withTimeout(promise, ms) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("timeout")), ms);
    promise.then((v) => { clearTimeout(timer); resolve(v); }, (e) => { clearTimeout(timer); reject(e); });
  });
}

async function trim(cache) {
  const keys = await cache.keys();
  for (let i = 0; i < keys.length - MAX_DATA; i += 1) { await cache.delete(keys[i]); }
}

async function networkFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  try {
    const response = await withTimeout(fetch(request), NETWORK_TIMEOUT_MS);
    if (response.ok) {
      // 지우고 다시 넣어 가장 최근으로(정리는 오래된 것부터).
      await cache.delete(request);
      await cache.put(request, response.clone());
      if (cacheName === DATA) { await trim(cache); }
    }
    return response;
  } catch (err) {
    const cached = await cache.match(request);
    if (cached) { return cached; }
    if (request.mode === "navigate") {
      const shell = await caches.match("/");
      if (shell) { return shell; }
    }
    return new Response(JSON.stringify({ error: { code: "OFFLINE", message: "offline" } }), {
      status: 503, headers: { "Content-Type": "application/json" },
    });
  }
}

async function cacheFirst(request) {
  const cached = await caches.match(request);
  if (cached) { return cached; }
  const response = await fetch(request);
  if (response.ok) { (await caches.open(SHELL)).put(request, response.clone()); }
  return response;
}

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") { return; }
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) { return; }
  if (request.mode === "navigate") {
    event.respondWith(networkFirst(request, SHELL));
    return;
  }
  if (url.pathname.startsWith("/assets/")) {
    event.respondWith(cacheFirst(request));
    return;
  }
  if (READABLE.some((re) => re.test(url.pathname))) {
    event.respondWith(networkFirst(request, DATA));
  }
});
