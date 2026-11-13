const dayFormat = new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium", timeZone: "Asia/Seoul" });

/**
 * 기록 메타의 날짜 "2026. 11. 13" — ko-KR 의 끝 마침표를 떼어 뒤의 구분점(" · ")과 겹치지 않게 한다.
 * 날짜가 아니면 "".
 */
export function historyDay(value: string | number | Date): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) { return ""; }
  return dayFormat.format(date).replace(/\.$/, "");
}
