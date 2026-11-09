/** 목록 항목을 "더 깊게" 원 강의 아래로 묶은 결과. */
export interface LectureGroup<T> {
  item: T;
  children: T[];
}

/**
 * 원 강의가 같은 목록에 있으면 심화 강의를 그 아래로 묶는다(목록 순서 유지, 여러 단계도 맨 위 원 강의로).
 * 원 강의가 목록에 없으면 그대로 최상위에 둔다.
 */
export function groupByParent<T>(
  items: T[],
  idOf: (item: T) => string,
  parentOf: (item: T) => string | undefined,
): LectureGroup<T>[] {
  const byId = new Map(items.map((item) => [idOf(item), item]));
  function rootOf(item: T): string {
    const seen = new Set<string>();
    let current = item;
    for (;;) {
      const parentId = parentOf(current);
      const parent = parentId ? byId.get(parentId) : undefined;
      if (!parent || seen.has(parentId ?? "")) { return idOf(current); }
      seen.add(parentId ?? "");
      current = parent;
    }
  }
  const groups = new Map<string, LectureGroup<T>>();
  const order: string[] = [];
  for (const item of items) {
    const root = rootOf(item);
    if (root === idOf(item)) {
      if (!groups.has(root)) { order.push(root); }
      groups.set(root, { item, children: groups.get(root)?.children ?? [] });
    } else {
      const rootItem = byId.get(root) as T;
      if (!groups.has(root)) {
        order.push(root);
        groups.set(root, { item: rootItem, children: [] });
      }
      groups.get(root)?.children.push(item);
    }
  }
  return order.map((id) => groups.get(id) as LectureGroup<T>);
}
