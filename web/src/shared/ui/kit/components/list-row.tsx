
import * as React from "react";
import { cn, focusRing } from "../lib/utils";

const ListContext = React.createContext(false);

export interface ListProps extends React.HTMLAttributes<HTMLUListElement> {
  /** 위·아래 바깥 구분선까지 그릴지 (기본: 행 사이만) */
  bordered?: boolean;
}

/** List — 상자(카드) 없이 얇은 구분선으로 나뉜 목록. ListRow의 부모. */
export function List({ bordered = false, className, ...props }: ListProps) {
  return (
    <ListContext.Provider value={true}>
      <ul role="list" className={cn(bordered && "border-y border-line", className)} {...props} />
    </ListContext.Provider>
  );
}

type PressableProps =
  | { href: string; linkAs?: React.ElementType; onClick?: React.MouseEventHandler; disabled?: never }
  | { href?: undefined; linkAs?: never; onClick?: React.MouseEventHandler; disabled?: boolean };

export type ListRowProps = Omit<React.HTMLAttributes<HTMLElement>, "title" | "onClick"> &
  PressableProps & {
    title: React.ReactNode;
    /** 둘째 줄: 출처·시간 등 한 줄 메타. 배지 대신 여기에 */
    meta?: React.ReactNode;
    leading?: React.ReactNode;
    /** 오른쪽: Badge 최대 1개, 또는 IconButton/Menu */
    trailing?: React.ReactNode;
    /** 선택(현재 열린 항목) — aria-current + 강조색 막대 */
    selected?: boolean;
    /** 읽지 않음 등 제목 굵기만 바꾸는 상태 */
    emphasized?: boolean;
    /** 제목 최대 줄 수 (기본 2) */
    titleLines?: 1 | 2;
    /** 메타 최대 줄 수 (기본 1: 한 줄 말줄임). 2면 두 줄까지 보이고 줄바꿈(\n)을 지킨다. "all"은 자르지 않는다 */
    metaLines?: 1 | 2 | "all";
  };

/**
 * ListRow — leading / title / meta / trailing 슬롯.
 * href → 링크(linkAs로 Next Link 등 주입), onClick → 버튼, 둘 다 없으면 정적 행.
 * trailing 안의 버튼은 행 클릭과 분리된다(행 전체가 버튼일 때 중첩 버튼을 만들지 않도록 trailing은 밖에 둔다).
 */
export function ListRow(props: ListRowProps) {
  const {
    title, meta, leading, trailing, selected = false, emphasized = false, titleLines = 2, metaLines = 1,
    href, linkAs, onClick, disabled, className, ...rest
  } = props;
  const inList = React.useContext(ListContext);
  const Outer = inList ? "li" : "div";

  const body = (
    <>
      {leading ? <span className="flex shrink-0 items-center text-fg-muted [&_svg]:h-5 [&_svg]:w-5">{leading}</span> : null}
      <span className="min-w-0 flex-1">
        <span
          className={cn(
            "block text-sm text-fg",
            emphasized ? "font-semibold" : "font-medium",
            titleLines === 1 ? "truncate" : "line-clamp-2"
          )}
        >
          {title}
        </span>
        {meta ? (
          <span
            className={cn(
              "mt-0.5 block text-xs text-fg-muted",
              metaLines === 1 ? "truncate" : metaLines === 2 ? "line-clamp-2 whitespace-pre-line break-keep" : "whitespace-pre-line break-keep",
            )}
          >
            {meta}
          </span>
        ) : null}
      </span>
    </>
  );

  const pressable = cn(
    "flex min-h-touch min-w-0 flex-1 items-center gap-3 py-3 text-left",
    "-mx-2 rounded-md px-2 transition-colors duration-fast hover:bg-canvas-hover",
    focusRing
  );

  let main: React.ReactNode;
  if (href !== undefined) {
    const Link = linkAs ?? "a";
    main = (
      <Link href={href} onClick={onClick} aria-current={selected ? "page" : undefined} className={pressable}>
        {body}
      </Link>
    );
  } else if (onClick) {
    main = (
      <button type="button" onClick={onClick} disabled={disabled} aria-current={selected || undefined}
        className={cn(pressable, "disabled:opacity-50")}>
        {body}
      </button>
    );
  } else {
    main = <div className="flex min-h-touch min-w-0 flex-1 items-center gap-3 py-3">{body}</div>;
  }

  return (
    <Outer
      className={cn(
        "relative flex items-center gap-2 border-b border-line last:border-b-0",
        selected && "before:absolute before:inset-y-2 before:-left-3 before:w-0.5 before:rounded-full before:bg-accent-solid before:content-['']",
        className
      )}
      {...rest}
    >
      {main}
      {trailing ? <span className="flex shrink-0 items-center gap-1">{trailing}</span> : null}
    </Outer>
  );
}
