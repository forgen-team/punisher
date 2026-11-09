// Small Next.js-style navigation API on top of React Router, so the Punisher
// widgets ported from the Witems web app keep their call sites
// (<Link href>, useRouter().push/replace, usePathname, useSearchParams).
import { forwardRef, useMemo, type AnchorHTMLAttributes } from "react";
import { Link as RouterLink, useLocation, useNavigate } from "react-router-dom";

export interface LinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href"> {
  href: string;
  replace?: boolean;
  /** Accepted for Next.js compatibility; ignored. */
  prefetch?: boolean;
  /** Accepted for Next.js compatibility; ignored. */
  scroll?: boolean;
}

/** In-app link. External (http…) links render a plain anchor. */
export const Link = forwardRef<HTMLAnchorElement, LinkProps>(function Link(
  { href, replace, prefetch: _prefetch, scroll: _scroll, ...rest },
  ref,
) {
  if (/^[a-z]+:/i.test(href)) {
    return <a ref={ref} href={href} {...rest} />;
  }
  return <RouterLink ref={ref} to={href} replace={replace} {...rest} />;
});

export interface AppRouter {
  push: (href: string, options?: { scroll?: boolean }) => void;
  replace: (href: string, options?: { scroll?: boolean }) => void;
  back: () => void;
}

export function useRouter(): AppRouter {
  const navigate = useNavigate();
  return useMemo(
    () => ({
      push: (href: string) => navigate(href),
      replace: (href: string) => navigate(href, { replace: true }),
      back: () => navigate(-1),
    }),
    [navigate],
  );
}

export function usePathname(): string {
  return useLocation().pathname;
}

export function useSearchParams(): URLSearchParams {
  const { search } = useLocation();
  return useMemo(() => new URLSearchParams(search), [search]);
}
