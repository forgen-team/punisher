export function lectureHref(id: string): string {
  return `/lectures/${encodeURIComponent(id)}`;
}
