// Package i18n holds the server-made learner-facing texts that are stored or
// streamed (i18n design L4): each has a stable code the web translates. The
// Korean text stays the stored value (generation_jobs.error,
// lectures.flag_reason) for older clients; responses add the codes.
package i18n

import "strings"

// Generation failure / review reason codes (StageEvent.reasonCodes, Lecture.flagReasonCodes).
const (
	ReasonGenerationError = "generation_error"
	ReasonTimeout         = "timeout"
	ReasonTruncated       = "truncated"
	ReasonFormat          = "format"
	ReasonVolume          = "volume"
	ReasonBlocks          = "blocks"
	ReasonLeak            = "leak"
	ReasonReview          = "review"
	ReasonUnreviewable    = "unreviewable"
	// reviewPrefix + review finding kind (factual_error, structure, …).
	reviewPrefix = "review."
)

// reasonKo is the stored Korean text of each code.
var reasonKo = map[string]string{
	ReasonGenerationError:              "강의를 만드는 중 오류가 났어요. 잠시 후 다시 만들어 주세요",
	ReasonTimeout:                      "강의를 쓰는 데 시간이 너무 오래 걸렸어요",
	ReasonTruncated:                    "강의가 길어져 끝까지 쓰지 못했어요",
	ReasonFormat:                       "강의 형식을 지키지 못했어요",
	ReasonVolume:                       "분량이 기준에 맞지 않았어요",
	ReasonBlocks:                       "정의·손계산·예측·연습·요약 구성이 기준에 못 미쳤어요",
	ReasonLeak:                         "생성 과정의 문구나 잘못된 링크가 남아 있었어요",
	ReasonReview:                       "검수 기준을 통과하지 못했어요",
	ReasonUnreviewable:                 "강의의 개념을 찾을 수 없어 검수하지 못했어요",
	reviewPrefix + "factual_error":     "사실과 다르거나 확인이 필요한 설명이 있었어요",
	reviewPrefix + "structure":         "핵심 정의나 논지 등 강의 구성에 문제가 있었어요",
	reviewPrefix + "pedagogy":          "손계산·예측·연습 같은 학습 활동이 부족했어요",
	reviewPrefix + "omission":          "학습 목표 일부를 충분히 다루지 못했어요",
	reviewPrefix + "assumed_knowledge": "설명 없이 쓰인 용어가 있었어요",
	reviewPrefix + "overclaim":         "근거 없이 단정한 문장이 있었어요",
	reviewPrefix + "source":            "근거 문서가 공식 문서가 아니었어요",
	reviewPrefix + "terminology":       "용어 정의가 일관되지 않았어요",
	reviewPrefix + "typo":              "실행되지 않는 코드나 오타가 있었어요",
}

var codeOfKo = func() map[string]string {
	m := make(map[string]string, len(reasonKo))
	for code, text := range reasonKo {
		m[text] = code
	}
	return m
}()

// ReasonKo is the Korean text of a reason code ("" when unknown).
func ReasonKo(code string) string { return reasonKo[code] }

// ReviewReasonCode is the code of a review finding kind (ReasonReview when the kind has no own reason).
func ReviewReasonCode(kind string) string {
	if _, ok := reasonKo[reviewPrefix+kind]; ok {
		return reviewPrefix + kind
	}
	return ReasonReview
}

// ReasonCodes maps stored reason texts back to their codes. It returns nil
// unless every text is a known reason (e.g. an LLM-written or legacy text),
// so clients either translate all of them or show the texts as they are.
func ReasonCodes(texts []string) []string {
	var out []string
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		code, ok := codeOfKo[t]
		if !ok {
			return nil
		}
		out = append(out, code)
	}
	return out
}

// FlagReasonSeparator joins the reasons of lectures.flag_reason.
const FlagReasonSeparator = "; "

// FlagReasonCodes is ReasonCodes of a stored lectures.flag_reason.
func FlagReasonCodes(flagReason string) []string {
	if strings.TrimSpace(flagReason) == "" {
		return nil
	}
	return ReasonCodes(strings.Split(flagReason, strings.TrimSpace(FlagReasonSeparator)))
}
