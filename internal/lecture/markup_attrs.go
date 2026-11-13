package lecture

import (
	"bytes"
	"encoding/json"
	"strings"
)

// parseAttrs parses `kind key=value flag ... title=rest of line`. The first
// bare token is stored under "_". title= and term= consume the rest of the line.
func parseAttrs(args string) map[string]string {
	attrs := map[string]string{}
	rest := strings.TrimSpace(args)
	for rest != "" {
		for _, greedy := range []string{"title=", "term="} {
			if strings.HasPrefix(rest, greedy) {
				attrs[strings.TrimSuffix(greedy, "=")] = strings.Trim(strings.TrimSpace(rest[len(greedy):]), `"`)
				return attrs
			}
		}
		token, tail, _ := strings.Cut(rest, " ")
		rest = strings.TrimSpace(tail)
		if key, value, ok := strings.Cut(token, "="); ok {
			attrs[key] = strings.Trim(value, `"`)
		} else {
			if _, seen := attrs["_"]; !seen {
				attrs["_"] = token
			}
			attrs[token] = ""
		}
	}
	return attrs
}

func parseSources(body string) []Source {
	out := []Source{}
	for _, line := range strings.Split(body, "\n") {
		if m := sourceRe.FindStringSubmatch(line); m != nil {
			out = append(out, Source{Title: strings.TrimSpace(m[1]), URL: m[2]})
		}
	}
	return out
}

func splitList(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// compactJSONObject parses a structured block body (an optional ```json fence
// is tolerated) into compact JSON; ok=false unless it is one JSON object.
func compactJSONObject(body string) (json.RawMessage, bool) {
	t := strings.TrimSpace(body)
	if strings.HasPrefix(t, "```") {
		if nl := strings.IndexByte(t, '\n'); nl >= 0 && strings.HasSuffix(t, "```") {
			t = strings.TrimSpace(strings.TrimSuffix(t[nl+1:], "```"))
		}
	}
	if !strings.HasPrefix(t, "{") {
		return nil, false
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(t)); err != nil {
		return nil, false
	}
	return json.RawMessage(buf.Bytes()), true
}
