// Small Next.js-style navigation API on top of React Router, so the Punisher
// widgets ported from the Witems web app keep their call sites
// (<Link href>, useRouter().push/replace, usePathname, useSearchParams).
import { forwardRef, useCallback, useEffect, useMemo, useRef, type AnchorHTMLAttributes, type MouseEvent } from "react";
import { Link as RouterLink, useLocation, useNavigate, useNavigationType } from "react-router-dom";
import { isWitems } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { confirmAction } from "@/src/shared/ui/kit";
import { kitMessages } from "@/src/shared/ui/kit/lib/messages";
import { isShellNavigation } from "./in-app-history";
import { hasUnsavedEdits } from "./leave-guard";

export interface LinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href"> {
  href: string;
  replace?: boolean;
  /** Accepted for Next.js compatibility; ignored. */
  prefetch?: boolean;
  /** Accepted for Next.js compatibility; ignored. */
  scroll?: boolean;
}

/** 저장하지 않은 편집이 있으면 앱 안 확인 창으로 묻는다(shared/lib/leave-guard). 떠나도 되면 true. */
export function useConfirmLeave(): () => Promise<boolean> {
  const t = useT(kitMessages);
  return useCallback(async () => {
    if (!hasUnsavedEdits()) { return true; }
    return confirmAction({ title: t.unsavedTitle, description: t.unsavedBody, confirmLabel: t.unsavedLeave });
  }, [t]);
}

function isPlainClick(event: MouseEvent<HTMLAnchorElement>): boolean {
  return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
}

/**
 * In-app link — a real <a href> (⌘/Ctrl·middle click open a new tab; inside Witems the href carries the
 * shell's basePath). External (http…) links render a plain anchor. A plain click while an editor has unsaved
 * edits asks first.
 */
export const Link = forwardRef<HTMLAnchorElement, LinkProps>(function Link(
  { href, replace, prefetch: _prefetch, scroll: _scroll, ...rest },
  ref,
) {
  if (/^[a-z]+:/i.test(href)) {
    return <a ref={ref} href={href} {...rest} />;
  }
  return <InAppLink ref={ref} href={href} replace={replace} {...rest} />;
});

const InAppLink = forwardRef<HTMLAnchorElement, Omit<LinkProps, "prefetch" | "scroll">>(function InAppLink(
  { href, replace, onClick, ...rest },
  ref,
) {
  const navigate = useNavigate();
  const confirmLeave = useConfirmLeave();
  return (
    <RouterLink
      ref={ref}
      to={href}
      replace={replace}
      onClick={(event) => {
        onClick?.(event);
        if (event.defaultPrevented || !isPlainClick(event) || !hasUnsavedEdits()) { return; }
        event.preventDefault();
        void confirmLeave().then((ok) => { if (ok) { navigate(href, { replace }); } });
      }}
      {...rest}
    />
  );
});

export interface NavigateOptions {
  scroll?: boolean;
  /** false: 저장하지 않은 편집이 있어도 묻지 않는다(같은 화면 안의 이동 — 초안은 보관된다) */
  guard?: boolean;
  /** 이동한 화면에서 이 스크롤 위치로 돌아간다(탭으로 돌아올 때) */
  restoreScroll?: number;
}

export interface AppRouter {
  push: (href: string, options?: NavigateOptions) => void;
  replace: (href: string, options?: NavigateOptions) => void;
  back: () => void;
}

/** 지금 주소와 같으면 이동하지 않는다(같은 값 push 는 기록만 늘린다). */
function sameLocation(href: string, pathname: string, search: string): boolean {
  return href === `${pathname}${search}` || (search === "" && href === pathname);
}

export function useRouter(): AppRouter {
  const navigate = useNavigate();
  const { pathname, search } = useLocation();
  const here = useRef({ pathname, search });
  here.current = { pathname, search };
  const confirmLeave = useConfirmLeave();
  return useMemo(
    () => {
      const go = (href: string, replace: boolean, options?: NavigateOptions) => {
        if (sameLocation(href, here.current.pathname, here.current.search)) { return; }
        const state = typeof options?.restoreScroll === "number" ? { restoreScroll: options.restoreScroll } : undefined;
        if (options?.guard === false || !hasUnsavedEdits()) {
          navigate(href, { replace, state });
          return;
        }
        void confirmLeave().then((ok) => { if (ok) { navigate(href, { replace, state }); } });
      };
      return {
        push: (href: string, options?: NavigateOptions) => go(href, false, options),
        replace: (href: string, options?: NavigateOptions) => go(href, true, options),
        // Witems 안에서는 주소의 주인이 셸이라 브라우저 뒤로가기를 쓴다(앱 라우터 기록은 직접 들어오면 비어 있다).
        back: () => (isWitems() ? window.history.back() : navigate(-1)),
      };
    },
    [navigate, confirmLeave],
  );
}

export function usePathname(): string {
  return useLocation().pathname;
}

export function useSearchParams(): URLSearchParams {
  const { search } = useLocation();
  return useMemo(() => new URLSearchParams(search), [search]);
}

/**
 * Next.js 처럼 다른 화면으로 가면(링크·push/replace) 맨 위에서 시작한다. 뒤로/앞으로(POP, Witems 는 셸이 알린
 * 이동)는 브라우저 복원에, #앵커·?at= 는 그 화면에 맡긴다(쿼리만 바뀌면 경로가 같아 움직이지 않는다).
 */
export function ScrollToTopOnNavigate() {
  const { pathname, hash, search, state, key } = useLocation();
  const type = useNavigationType();
  const restore = typeof state === "object" && state !== null ? (state as { restoreScroll?: unknown }).restoreScroll : undefined;
  const skip = type === "POP" || isShellNavigation(state) || Boolean(hash) || new URLSearchParams(search).has("at");
  const latest = useRef({ skip, restore, pathname });
  const prevPath = useRef(pathname);
  latest.current = { skip, restore, pathname };
  useEffect(() => {
    const { skip: keep, restore: y, pathname: path } = latest.current;
    const pathChanged = prevPath.current !== path;
    prevPath.current = path;
    if (typeof y === "number") {
      // 탭으로 돌아온 화면: 그 탭에서 마지막으로 보던 자리(내용이 그려진 다음 프레임에).
      const frame = requestAnimationFrame(() => window.scrollTo(0, y));
      return () => cancelAnimationFrame(frame);
    }
    // 쿼리만 바뀐 이동(필터·시트)은 자리를 지킨다.
    if (pathChanged && !keep) { window.scrollTo(0, 0); }
    return undefined;
  }, [key]);
  return null;
}
