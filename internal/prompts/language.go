package prompts

// OutputLanguageRule is appended after the (possibly user-edited) system
// prompt of every generation (i18n §3): Korean is the default and needs no
// rule; English overrides any language wording in the blocks above.
func OutputLanguageRule(locale string) string {
	if locale != "en" {
		return ""
	}
	return `

## 출력 언어: English (이 지시가 위의 언어·말투 관련 지시보다 우선합니다)
- 학습자에게 보일 모든 글 — 강의 본문·제목·요약·문항·채점 기준(must)·모범답·피드백·튜터 문장·용어 설명·해설 — 을 자연스럽고 정확한 영어로 씁니다. 실무 개발자에게 쓰는 담백한 기술 영어(평서문, 과장 없음).
- 위 지시의 '존댓말·해요체'는 정중하고 직접적인 영어 어조로, '…자 이내' 같은 글자 수 기준은 영어로 같은 내용을 담는 분량(약 2배 글자 수)으로 바꿔 적용합니다.
- 바꾸지 않는 것: JSON 키와 enum 값, @@ 지시어와 그 속성(kind=, os= 등), id·slug·코드·명령어·로그. 실습 블록의 표지는 'Look:'·'Expect:' 를 씁니다.
- 아래 자료(개념 이름·목표 등)가 한국어여도 출력은 영어로 씁니다.`
}

// WithOutputLanguage appends OutputLanguageRule to a system prompt.
func WithOutputLanguage(system, locale string) string {
	return system + OutputLanguageRule(locale)
}
