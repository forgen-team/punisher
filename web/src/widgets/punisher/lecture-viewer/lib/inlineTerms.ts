/**
 * 예전 강의는 모델이 문장 안에 "@@term concept=os.thread term=스레드 들은…"처럼 지시어를 그대로 썼다
 * (@@ 는 줄 맨 앞 전용이라 파서가 처리하지 않음). 이미 저장된 강의를 위해 렌더러에서 문장을 복구한다.
 * - concept+term → 인라인 링크 [스레드](term:os.thread)
 * - term 만 있거나 @@term{이름} → 그 용어를 평문으로
 * - 이름 없이 뒤에 낱말이 오면("@@term 배열의") 지시어만 지운다
 * - 이름이 아예 없으면("@@term 에서 다루는") 빈칸이 되지 않게 '관련 개념'으로 채운다(2026-11-10)
 *
 * 용어 끝은 다음 중 하나 앞에서 끊는다: 공백 뒤 조사/연결어로 시작하는 말(에서·와·과·을·를·은·는·이·가·의·로·으로·도·만·강의·들),
 * 문장 부호, 줄 끝, 또는 다음 @@term.
 */
const TERM_END = String.raw`(?=\s(?:에서|와|과|을|를|은|는|이|가|의|로|으로|도|만|강의|들)|\s*[,.:;)]|\s*$|\s+@@term)`;
const LEGACY_LINK = new RegExp(String.raw`@@term\s+concept=([\w.-]+)\s+(?:proposed\s+)?term=(.+?)${TERM_END}`, "gm");
const LEGACY_NAME_ONLY = new RegExp(String.raw`@@term\s+(?:proposed\s+)?term=(.+?)${TERM_END}`, "gm");
const LEGACY_BRACE = /@@term\{([^}\n]*)\}/g;
const LEGACY_BARE = /@@term\b(?:[^\S\n]+(?:concept=[\w.-]+|proposed)(?=\s|$))*[^\S\n]*(\S*)/g;

const PLACEHOLDER = "관련 개념";
/** '관련 개념'은 받침으로 끝나므로 받침 없는 말에 붙는 조사를 맞춰 준다. */
const CONSONANT_PARTICLE: Record<string, string> = {
  를: "을", 는: "은", 가: "이", 와: "과", 와의: "과의", 로: "으로", 로는: "으로는", 로서: "으로서", 로써: "으로써", 라는: "이라는", 라고: "이라고", 랑: "이랑", 나: "이나",
};
const PARTICLE = /^(에서는|에서도|에서|에게|에는|에도|에|과의|와의|과|와|을|를|은|는|입니다|이다|이고|이며|이나|이랑|이라는|이라고|이|가|의|으로서|으로써|으로는|으로|로서|로써|로는|로|도|만|나|랑|라는|라고|부터|까지|처럼|보다)(?=$|[^\p{L}\p{N}])/u;

function repairBare(_match: string, next: string): string {
  const particle = PARTICLE.exec(next)?.[1];
  if (particle) {
    return `${PLACEHOLDER}${CONSONANT_PARTICLE[particle] ?? particle}${next.slice(particle.length)}`;
  }
  return /^[\p{L}\p{N}]/u.test(next) || next.startsWith("[") ? next : `${PLACEHOLDER}${next}`;
}

export function toInlineTermLinks(text: string): string {
  return text
    .replace(LEGACY_LINK, (_m, id: string, term: string) => `[${term.trim()}](term:${id})`)
    .replace(LEGACY_NAME_ONLY, (_m, term: string) => term.trim())
    .replace(LEGACY_BRACE, (_m, term: string) => term.trim() || "@@term ")
    .replace(LEGACY_BARE, repairBare);
}
