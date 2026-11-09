// Package llmjson parses JSON payloads out of LLM responses.
//
// LLM output frequently wraps JSON in markdown code fences or surrounds it
// with prose ("Here is the JSON: ..."). TryParseJSON strips fences and, if the
// cleaned text is still not valid JSON, falls back to the outermost {…} or […]
// span. Logic originates from stock-server's services.TryParseJSON.
package llmjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrEmpty is returned when the content is blank after cleanup.
var ErrEmpty = errors.New("llmjson: empty content")

// TryParseJSON parses content into T, tolerating markdown fences and
// surrounding prose.
func TryParseJSON[T any](content string) (T, error) {
	var result T
	cleaned := StripMarkdownCodeBlock(content)
	if cleaned == "" {
		return result, ErrEmpty
	}

	firstErr := json.Unmarshal([]byte(cleaned), &result)
	if firstErr == nil {
		return result, nil
	}

	// A bare JSON object whose string values contain markdown code blocks
	// (e.g. {"answer": "...```kotlin\n...```"}) is mangled by fence stripping,
	// so the original text and its spans are tried as well.
	original := strings.TrimSpace(strings.TrimPrefix(content, "\xef\xbb\xbf"))
	candidates := candidateSpans(cleaned)
	if original != cleaned {
		candidates = append([]string{original}, candidates...)
		candidates = append(candidates, candidateSpans(original)...)
	}
	for _, span := range candidates {
		var candidate T
		if err := json.Unmarshal([]byte(span), &candidate); err == nil {
			return candidate, nil
		}
	}
	return result, fmt.Errorf("llmjson: failed to parse JSON: %w", firstErr)
}

// candidateSpans returns the outermost object and array spans, ordered so the
// span that starts earlier in the text is tried first.
func candidateSpans(s string) []string {
	objStart, objEnd := strings.Index(s, "{"), strings.LastIndex(s, "}")
	arrStart, arrEnd := strings.Index(s, "["), strings.LastIndex(s, "]")

	var spans []string
	hasObj := objStart >= 0 && objEnd > objStart
	hasArr := arrStart >= 0 && arrEnd > arrStart
	if hasArr && (!hasObj || arrStart < objStart) {
		spans = append(spans, s[arrStart:arrEnd+1])
	}
	if hasObj {
		spans = append(spans, s[objStart:objEnd+1])
	}
	if hasArr && hasObj && objStart < arrStart {
		spans = append(spans, s[arrStart:arrEnd+1])
	}
	return spans
}

// StripMarkdownCodeBlock removes a BOM and a surrounding (or embedded)
// ``` / ```json fence, returning the trimmed inner text.
func StripMarkdownCodeBlock(content string) string {
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "```") {
		if idx := strings.Index(content[3:], "\n"); idx >= 0 {
			content = content[3+idx+1:]
		} else {
			content = content[3:]
		}
		content = strings.TrimSpace(content)
		content = strings.TrimSuffix(content, "```")
		return strings.TrimSpace(content)
	}

	fenceStart := strings.Index(content, "```")
	if fenceStart < 0 {
		return content
	}
	inner := content[fenceStart+3:]
	if nl := strings.Index(inner, "\n"); nl >= 0 {
		inner = inner[nl+1:]
	}
	if fenceEnd := strings.Index(inner, "```"); fenceEnd >= 0 {
		return strings.TrimSpace(inner[:fenceEnd])
	}
	return strings.TrimSpace(inner)
}
