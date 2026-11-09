/**
 * 예전 강의는 모델이 문장 안에 "@@term concept=os.thread term=스레드 들은…"처럼 지시어를 그대로 썼다
 * (@@ 는 줄 맨 앞 전용이라 파서가 처리하지 않음). 이를 새 인라인 링크 [스레드](term:os.thread) 로 바꾸고,
 * 개념 id 없는 "@@term 배열" 은 지시어만 지운다.
 *
 * 용어 끝은 다음 중 하나 앞에서 끊는다: 공백 뒤 조사/연결어로 시작하는 말(에서·와·과·을·를·은·는·이·가·의·로·으로·도·만·강의·들),
 * 문장 부호, 줄 끝, 또는 다음 @@term.
 */
const LEGACY_LINK = /@@term\s+concept=([\w.-]+)\s+term=(.+?)(?=\s(?:에서|와|과|을|를|은|는|이|가|의|로|으로|도|만|강의|들)|\s*[,.:;)]|\s*$|\s+@@term)/gm;
const LEGACY_BARE = /@@term\s+(?!concept=)/g;

export function toInlineTermLinks(text: string): string {
  return text.replace(LEGACY_LINK, (_m, id: string, term: string) => `[${term.trim()}](term:${id})`).replace(LEGACY_BARE, "");
}
