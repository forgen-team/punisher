package i18n

import (
	"reflect"
	"testing"
)

func TestReasonCodesRoundTrip(t *testing.T) {
	for code := range reasonKo {
		if got := ReasonCodes([]string{ReasonKo(code)}); !reflect.DeepEqual(got, []string{code}) {
			t.Fatalf("%s: %v", code, got)
		}
	}
	if got := ReasonCodes([]string{ReasonKo(ReasonFormat), "모델이 쓴 사유"}); got != nil {
		t.Fatalf("an unknown text must leave the codes out: %v", got)
	}
	if got := ReviewReasonCode("unknown_kind"); got != ReasonReview {
		t.Fatalf("unknown kind: %s", got)
	}
}

func TestFlagReasonCodes(t *testing.T) {
	stored := ReasonKo("review.factual_error") + FlagReasonSeparator + ReasonKo(ReasonVolume)
	if got := FlagReasonCodes(stored); !reflect.DeepEqual(got, []string{"review.factual_error", ReasonVolume}) {
		t.Fatalf("codes: %v", got)
	}
	if FlagReasonCodes("") != nil {
		t.Fatal("empty flag reason")
	}
}
