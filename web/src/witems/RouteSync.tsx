import { useEffect, useRef } from "react";
import { useLocation, useNavigate, useNavigationType } from "react-router-dom";
import type { AppContext } from "./contract";

/**
 * Keeps the app's MemoryRouter and the shell URL the same (contract §3: the
 * shell owns the address bar). In-app navigation → ctx.navigate; back/forward
 * in the shell → ctx.onRouteChange → the memory router follows (replace).
 * Equal paths are skipped both ways, so neither side echoes the other.
 */
export function RouteSync({ ctx }: { ctx: AppContext }) {
  const location = useLocation();
  const navigate = useNavigate();
  const type = useNavigationType();
  const first = useRef(true);
  const current = useRef("");
  current.current = `${location.pathname}${location.search}`;

  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    const target = `${location.pathname}${location.search}`;
    if (target === `${ctx.path}${ctx.search}`) { return; }
    ctx.navigate(target, { replace: type === "REPLACE" });
  }, [ctx, location.pathname, location.search, type]);

  useEffect(
    () =>
      ctx.onRouteChange((path, search) => {
        // 앱이 ctx.navigate 로 바꾼 주소도 여기로 돌아온다 — 이미 그 자리면 다시 이동하지 않는다
        // (불필요한 REPLACE 가 쿼리만 바뀐 이동을 맨 위로 올리지 않게).
        if (`${path}${search}` === current.current) { return; }
        navigate(`${path}${search}`, { replace: true });
      }),
    [ctx, navigate],
  );

  return null;
}
