package quality

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/forgen-team/punisher/internal/lecture"
)

// ArithResult is one written calculation ("a op b = c") that does not hold.
type ArithResult struct {
	Anchor   string `json:"anchor"`
	Expr     string `json:"expr"`
	Stated   string `json:"stated"`
	Computed string `json:"computed"`
}

// CheckArithmetic recomputes the "expression = number" calculations of the
// lecture prose, answers and trace cells (code and inline code excluded) and
// returns the ones that do not hold. Accepted: rounding to the written
// decimals, base-2 sums of binary literals (also wrapped to the result's
// width), and 8/16/32/64-bit wrap-around. Anything that does not parse is skipped.
func CheckArithmetic(c lecture.Content) (mismatches []ArithResult, checked int) {
	var out []ArithResult
	scan := func(anchor, text string) {
		for _, e := range findEquations(text) {
			checked++
			if r, ok := e.verify(); !ok {
				out = append(out, ArithResult{Anchor: anchor, Expr: e.expr, Stated: e.result, Computed: r})
			}
		}
	}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			switch b.Type {
			case lecture.BlockMarkdown, lecture.BlockKeypoint, lecture.BlockExercise, lecture.BlockAnswer,
				lecture.BlockMisconception, lecture.BlockSummary:
				scan(b.Anchor, inlineCodeRe.ReplaceAllString(lecture.ProseOnly(b.Text), " "))
			case lecture.BlockTrace:
				var d lecture.TraceData
				if json.Unmarshal(b.Data, &d) == nil {
					scan(b.Anchor, d.Caption)
					scan(b.Anchor, d.Note)
					for _, row := range d.Rows {
						for _, cell := range row {
							scan(b.Anchor, inlineCodeRe.ReplaceAllString(cell, " "))
						}
					}
				}
			}
		}
	}
	return out, checked
}

type equation struct{ expr, result string }

// exprChar: characters an arithmetic left side may hold.
func exprChar(r rune) bool {
	return (r >= '0' && r <= '9') || strings.ContainsRune(" .,+-−×*/÷^()", r)
}

// findEquations finds "left = number" where left is only numbers, spaces,
// + - − × * / ÷ ^ ( ) and " x " (multiplication with spaces), with at least one operator.
func findEquations(text string) []equation {
	r := []rune(text)
	var out []equation
	for i, ch := range r {
		if ch != '=' || (i > 0 && strings.ContainsRune("=!<>≠", r[i-1])) || (i+1 < len(r) && r[i+1] == '=') {
			continue
		}
		// left side: scan back over expression characters (", " and ". " end it: list and sentence breaks).
		j := i
		for j > 0 {
			p := r[j-1]
			if p == 'x' && j >= 2 && r[j-2] == ' ' && j < len(r) && r[j] == ' ' {
				j--
				continue
			}
			if !exprChar(p) || ((p == ',' || p == '.') && j < len(r) && r[j] == ' ') {
				break
			}
			j--
		}
		left := strings.TrimSpace(string(r[j:i]))
		left = strings.TrimLeft(left, " ,.)+*/÷^×")
		// right side: one number, not followed by another operator.
		k := i + 1
		for k < len(r) && r[k] == ' ' {
			k++
		}
		start := k
		if k < len(r) && (r[k] == '-' || r[k] == '−') {
			k++
		}
		for k < len(r) && ((r[k] >= '0' && r[k] <= '9') || ((r[k] == ',' || r[k] == '.') && k+1 < len(r) && r[k+1] >= '0' && r[k+1] <= '9')) {
			k++
		}
		right := string(r[start:k])
		rest := strings.TrimLeft(string(r[k:min(k+3, len(r))]), " ")
		if right == "" || right == "-" || right == "−" || (rest != "" && strings.ContainsRune("+-−×*/÷^=", []rune(rest)[0])) {
			continue
		}
		if !hasOperator(left) || !strings.ContainsAny(left, "0123456789") {
			continue
		}
		out = append(out, equation{expr: left, result: right})
	}
	return out
}

func hasOperator(s string) bool {
	t := strings.TrimLeft(s, "-−( ")
	return strings.ContainsAny(t, "+-−×*/÷^") || strings.Contains(t, " x ")
}

// verify recomputes e; ok when it holds under one of the accepted readings.
func (e equation) verify() (string, bool) {
	want, err := parseNumber(e.result)
	if err != nil {
		return "", true
	}
	got, err := evalExpr(e.expr, 10)
	if err != nil {
		return "", true // not something we can read: skip
	}
	decimals := 0
	if dot := strings.IndexByte(strings.ReplaceAll(e.result, ",", ""), '.'); dot >= 0 {
		decimals = len(strings.ReplaceAll(e.result, ",", "")) - dot - 1
	}
	if closeTo(got, want, decimals) || wraps(got, want) {
		return "", true
	}
	if binaryLiterals(e.expr, e.result) {
		if b, err := evalExpr(e.expr, 2); err == nil {
			w, _ := strconv.ParseInt(strings.TrimLeft(e.result, "-−"), 2, 64)
			mod := math.Pow(2, float64(len(e.result)))
			if b == float64(w) || math.Mod(math.Mod(b, mod)+mod, mod) == float64(w) {
				return "", true
			}
		}
		return "", true // binary-looking but not a plain base-2 sum: skip
	}
	return formatNumber(got), false
}

func closeTo(got, want float64, decimals int) bool {
	tol := 0.5 * math.Pow(10, -float64(decimals))
	if decimals == 0 {
		tol = 0.5 // rounding and "7 / 2 = 3" (integer division) both pass
	}
	return math.Abs(got-want) <= tol*(1+1e-9) || math.Abs(got-want) <= 1e-12*math.Max(1, math.Abs(want))
}

// wraps: got and want are the same value in an 8/16/32/64-bit register
// (signed or unsigned), compared as exact integers.
func wraps(got, want float64) bool {
	if got != math.Trunc(got) || want != math.Trunc(want) || got == want {
		return false
	}
	g, _ := new(big.Float).SetFloat64(got).Int(nil)
	w, _ := new(big.Float).SetFloat64(want).Int(nil)
	for _, bits := range []uint{8, 16, 32, 64} {
		mod := new(big.Int).Lsh(big.NewInt(1), bits)
		if new(big.Int).Mod(g, mod).Cmp(new(big.Int).Mod(w, mod)) == 0 {
			return true
		}
	}
	return false
}

func binaryLiterals(expr, result string) bool {
	for _, tok := range strings.FieldsFunc(expr+" "+result, func(r rune) bool { return !(r >= '0' && r <= '9') }) {
		if strings.Trim(tok, "01") != "" {
			return false
		}
	}
	return true
}

func parseNumber(s string) (float64, error) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, ",", ""), "−", "-")
	return strconv.ParseFloat(s, 64)
}

func formatNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e18 {
		return strconv.FormatFloat(f, 'f', 0, 64)
	}
	return strconv.FormatFloat(f, 'g', 12, 64)
}

// evalExpr evaluates + - * / ^ and parentheses (× ÷ − and " x " accepted);
// numbers are read in base (2 or 10).
func evalExpr(s string, base int) (float64, error) {
	s = strings.NewReplacer("×", "*", "÷", "/", "−", "-", " x ", "*").Replace(s)
	p := &exprParser{s: strings.ReplaceAll(s, " ", ""), base: base}
	v, err := p.sum()
	if err != nil {
		return 0, err
	}
	if p.i != len(p.s) {
		return 0, fmt.Errorf("unexpected %q", p.s[p.i:])
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, fmt.Errorf("not finite")
	}
	return v, nil
}

type exprParser struct {
	s    string
	i    int
	base int
}

func (p *exprParser) peek() byte {
	if p.i < len(p.s) {
		return p.s[p.i]
	}
	return 0
}

func (p *exprParser) sum() (float64, error) {
	v, err := p.product()
	for err == nil && (p.peek() == '+' || p.peek() == '-') {
		op := p.peek()
		p.i++
		var r float64
		if r, err = p.product(); op == '+' {
			v += r
		} else {
			v -= r
		}
	}
	return v, err
}

func (p *exprParser) product() (float64, error) {
	v, err := p.power()
	for err == nil && (p.peek() == '*' || p.peek() == '/') {
		op := p.peek()
		p.i++
		var r float64
		if r, err = p.power(); err == nil {
			if op == '*' {
				v *= r
			} else if r == 0 {
				return 0, fmt.Errorf("division by zero")
			} else {
				v /= r
			}
		}
	}
	return v, err
}

func (p *exprParser) power() (float64, error) {
	v, err := p.unary()
	if err == nil && p.peek() == '^' {
		p.i++
		var e float64
		if e, err = p.power(); err == nil {
			v = math.Pow(v, e)
		}
	}
	return v, err
}

func (p *exprParser) unary() (float64, error) {
	if p.peek() == '-' {
		p.i++
		v, err := p.unary()
		return -v, err
	}
	if p.peek() == '(' {
		p.i++
		v, err := p.sum()
		if err != nil || p.peek() != ')' {
			return 0, fmt.Errorf("unbalanced parenthesis")
		}
		p.i++
		return v, nil
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.' || p.s[p.i] == ',') {
		p.i++
	}
	tok := strings.ReplaceAll(p.s[start:p.i], ",", "")
	if tok == "" {
		return 0, fmt.Errorf("number expected at %d", start)
	}
	if p.base == 2 {
		n, err := strconv.ParseInt(tok, 2, 64)
		return float64(n), err
	}
	return strconv.ParseFloat(tok, 64)
}

// ruleArithmetic (G22): every recomputed calculation holds.
func ruleArithmetic(results []ArithResult, checked int) RuleResult {
	if checked == 0 {
		return na("G22", "다시 계산할 식 없음")
	}
	bad := []string{fmt.Sprintf("식 %d개 중 틀림 %d개", checked, len(results))}
	for i, r := range results {
		if i == 4 {
			bad = append(bad, "…")
			break
		}
		bad = append(bad, fmt.Sprintf("%s: %s = %s (계산하면 %s)", r.Anchor, r.Expr, r.Stated, r.Computed))
	}
	return result("G22", len(results) == 0, float64(len(results)), "틀린 계산 0", strings.Join(bad, "; "))
}
