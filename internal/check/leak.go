package check

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxLeakRatio: a text sharing more than this share of the answer's word
// 3-grams gives the answer away. A secondary guard only — the real defence is
// that any answer after narrowing counts as aided (design §4.2).
const MaxLeakRatio = 0.3

// endings are particles/verb endings stripped from each word (longest first)
// so "캐시는" and "캐시가" compare equal.
var endings = func() []string {
	list := []string{
		"입니다", "합니다", "됩니다", "이에요", "예요", "해요", "돼요", "에서", "으로", "에게", "한테", "까지", "부터", "처럼", "보다",
		"이다", "하다", "되다", "다고", "라고", "지만", "고", "며", "면", "은", "는", "이", "가", "을", "를", "에", "로", "와", "과", "의", "도", "만", "다", "요",
	}
	sort.SliceStable(list, func(i, j int) bool { return utf8.RuneCountInString(list[i]) > utf8.RuneCountInString(list[j]) })
	return list
}()

// Normalize lowercases, drops punctuation and strips one particle/ending per word.
func Normalize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, stripEnding(f))
	}
	return out
}

func stripEnding(word string) string {
	for _, e := range endings {
		if strings.HasSuffix(word, e) && utf8.RuneCountInString(word) > utf8.RuneCountInString(e) {
			return strings.TrimSuffix(word, e)
		}
	}
	return word
}

func trigrams(words []string) map[string]bool {
	out := map[string]bool{}
	for i := 0; i+3 <= len(words); i++ {
		out[words[i]+"\x00"+words[i+1]+"\x00"+words[i+2]] = true
	}
	return out
}

// LeakRatio is the share of the answer's word 3-grams that also appear in text
// (0 when the answer has fewer than three words).
func LeakRatio(text, answer string) float64 {
	ans := trigrams(Normalize(answer))
	if len(ans) == 0 {
		return 0
	}
	got := trigrams(Normalize(text))
	shared := 0
	for g := range ans {
		if got[g] {
			shared++
		}
	}
	return float64(shared) / float64(len(ans))
}

// Leaks reports whether text gives the answer away.
func Leaks(text, answer string) bool { return LeakRatio(text, answer) > MaxLeakRatio }

// praiseRe matches stock praise ("좋은 질문이에요", "훌륭해요" …). Plain
// confirmations ("맞아요") are feedback, not praise, and stay.
var praiseRe = regexp.MustCompile(`(?i)(아주|정말|너무)?\s*(좋은\s*(질문|답변|답|시도|생각)(이에요|입니다|이네요|이에요|이군요)|훌륭(해요|합니다|하네요|하십니다)|잘\s*(했어요|하셨어요|하셨습니다|했습니다|하고\s*계세요)|완벽(해요|합니다|하네요)|멋져요|멋집니다|대단(해요|합니다|하네요)|great\s+(job|work|question)|excellent|well\s+done)[!.~]*`)

var spacesRe = regexp.MustCompile(`[ \t]{2,}`)

// StripPraise removes stock praise phrases and the leftover separators.
func StripPraise(text string) string {
	out := praiseRe.ReplaceAllString(text, "")
	out = spacesRe.ReplaceAllString(out, " ")
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimLeft(strings.TrimSpace(l), "!.,~ ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
