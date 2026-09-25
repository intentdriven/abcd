// Package actionsexpr evaluates GitHub Actions workflow expressions for tests
// that must ask what GitHub would DO with a committed workflow, not merely
// whether a string is present in it. A substring assertion passes a predicate
// that has been rewritten to mean something else; evaluating it cannot.
//
// It is test support, imported only by _test.go files, on the internal/gittest
// and internal/testsecret precedent: the workflow contracts live in more than
// one package (core/lint holds the repository's own workflows, and
// core/launch/scaffold the templates a managed repository is scaffolded from),
// and an evaluator spelled twice is one the two can disagree about.
//
// It implements the slice of the expression language the workflows here use,
// and it fails CLOSED on the rest: an unknown context path or an unsupported
// function is an error the caller reports as a failure, never a guessed value.
package actionsexpr

import (
	"fmt"
	"strconv"
	"strings"
)

// EvalValue evaluates one YAML value the way GitHub would, in the given
// expression context: a bare `true`/`false`, a value that is exactly one
// `${{ }}` expression (which keeps its own type), or a string with expressions
// interpolated into it.
func EvalValue(raw string, ctx map[string]any) (any, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, nil
	}
	if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
		v = v[1 : len(v)-1]
	}
	switch v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	start := strings.Index(v, "${{")
	if start < 0 {
		return v, nil
	}
	var sb strings.Builder
	rest := v
	whole := true
	for {
		start = strings.Index(rest, "${{")
		if start < 0 {
			sb.WriteString(rest)
			break
		}
		end := strings.Index(rest[start:], "}}")
		if end < 0 {
			return nil, fmt.Errorf("unterminated ${{ ... }} in %q", raw)
		}
		end += start
		val, err := evalExpr(rest[start+3:end], ctx)
		if err != nil {
			return nil, err
		}
		if start == 0 && end+2 == len(rest) && sb.Len() == 0 && whole {
			return val, nil
		}
		whole = false
		sb.WriteString(rest[:start])
		sb.WriteString(Stringify(val))
		rest = rest[end+2:]
	}
	return sb.String(), nil
}

// The expression evaluator below implements the slice of GitHub's expression
// language the workflows here use: literals, context lookups, the comparison
// and logical operators, four string functions, and the four status-check
// functions a job condition reads. It is
// deliberately partial and deliberately strict — an unknown context path or an
// unsupported function is an ERROR, which the caller reports as a violation.
// The alternative, guessing a value, would let this gate pass a block it does
// not understand, and the whole class of defect here is one that passes
// unnoticed.

type exprToken struct {
	kind string // "op", "str", "num", "ident"
	text string
}

type exprParser struct {
	toks []exprToken
	pos  int
	ctx  map[string]any
}

func evalExpr(src string, ctx map[string]any) (any, error) {
	toks, err := tokenizeExpr(src)
	if err != nil {
		return nil, fmt.Errorf("in `%s`: %w", strings.TrimSpace(src), err)
	}
	p := &exprParser{toks: toks, ctx: ctx}
	v, err := p.parseOr()
	if err != nil {
		return nil, fmt.Errorf("in `%s`: %w", strings.TrimSpace(src), err)
	}
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("in `%s`: unexpected %q", strings.TrimSpace(src), p.toks[p.pos].text)
	}
	return v, nil
}

func tokenizeExpr(s string) ([]exprToken, error) {
	isIdent := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '_' || c == '-' || c == '.'
	}
	var out []exprToken
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '\'':
			var sb strings.Builder
			j := i + 1
			closed := false
			for j < len(s) {
				if s[j] == '\'' {
					if j+1 < len(s) && s[j+1] == '\'' { // '' is an escaped quote
						sb.WriteByte('\'')
						j += 2
						continue
					}
					closed = true
					break
				}
				sb.WriteByte(s[j])
				j++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated string literal")
			}
			out = append(out, exprToken{"str", sb.String()})
			i = j + 1
		case strings.HasPrefix(s[i:], "=="), strings.HasPrefix(s[i:], "!="),
			strings.HasPrefix(s[i:], "&&"), strings.HasPrefix(s[i:], "||"):
			out = append(out, exprToken{"op", s[i : i+2]})
			i += 2
		case c == '(' || c == ')' || c == ',' || c == '!':
			out = append(out, exprToken{"op", string(c)})
			i++
		case c >= '0' && c <= '9':
			j := i
			for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
				j++
			}
			out = append(out, exprToken{"num", s[i:j]})
			i = j
		case isIdent(c):
			j := i
			for j < len(s) && isIdent(s[j]) {
				j++
			}
			out = append(out, exprToken{"ident", s[i:j]})
			i = j
		default:
			return nil, fmt.Errorf("unsupported character %q", string(c))
		}
	}
	return out, nil
}

func (p *exprParser) peek() (exprToken, bool) {
	if p.pos >= len(p.toks) {
		return exprToken{}, false
	}
	return p.toks[p.pos], true
}

func (p *exprParser) acceptOp(op string) bool {
	if t, ok := p.peek(); ok && t.kind == "op" && t.text == op {
		p.pos++
		return true
	}
	return false
}

// GitHub's `||` and `&&` yield an OPERAND, not a boolean: `a || b` is a when a
// is truthy and b otherwise. That is what makes `${{ x || github.run_id }}` a
// group key rather than the word "true", so the evaluator must reproduce it.
func (p *exprParser) parseOr() (any, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.acceptOp("||") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		if !Truthy(left) {
			left = right
		}
	}
	return left, nil
}

func (p *exprParser) parseAnd() (any, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.acceptOp("&&") {
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		if Truthy(left) {
			left = right
		}
	}
	return left, nil
}

func (p *exprParser) parseEquality() (any, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case p.acceptOp("=="):
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = looseEqual(left, right)
		case p.acceptOp("!="):
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = !looseEqual(left, right)
		default:
			return left, nil
		}
	}
}

func (p *exprParser) parseUnary() (any, error) {
	if p.acceptOp("!") {
		v, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return !Truthy(v), nil
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (any, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("expression ends early")
	}
	p.pos++
	switch t.kind {
	case "op":
		if t.text == "(" {
			v, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			if !p.acceptOp(")") {
				return nil, fmt.Errorf("missing )")
			}
			return v, nil
		}
		return nil, fmt.Errorf("unexpected %q", t.text)
	case "str":
		return t.text, nil
	case "num":
		n, err := strconv.ParseFloat(t.text, 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", t.text)
		}
		return n, nil
	}

	if next, ok := p.peek(); ok && next.kind == "op" && next.text == "(" {
		p.pos++
		var args []any
		if !p.acceptOp(")") {
			for {
				a, err := p.parseOr()
				if err != nil {
					return nil, err
				}
				args = append(args, a)
				if p.acceptOp(",") {
					continue
				}
				if p.acceptOp(")") {
					break
				}
				return nil, fmt.Errorf("missing ) after %s(", t.text)
			}
		}
		return callFunc(t.text, args, p.ctx)
	}

	switch strings.ToLower(t.text) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}
	v, ok := p.ctx[t.text]
	if !ok {
		return nil, fmt.Errorf("unknown context path %q; add it to the context the caller "+
			"hands in, with the value that event carries", t.text)
	}
	return v, nil
}

// statusFuncs are the status-check functions. Their value is the state of the
// RUN, not of any context property, so the caller states it: a context that
// means to answer cancelled() carries the key "cancelled()". always() needs no
// statement. A status function the context does not answer is an error.
var statusFuncs = map[string]bool{"success": true, "failure": true, "cancelled": true, "always": true}

func callFunc(name string, args []any, ctx map[string]any) (any, error) {
	lower := strings.ToLower(name)
	if statusFuncs[lower] {
		if len(args) != 0 {
			return nil, fmt.Errorf("%s() takes no arguments", name)
		}
		if lower == "always" {
			return true, nil
		}
		v, ok := ctx[lower+"()"]
		if !ok {
			return nil, fmt.Errorf("%s() is not stated in the context; add the key %q with the "+
				"run state being asked about", name, lower+"()")
		}
		return v, nil
	}
	switch lower {
	case "format":
		if len(args) == 0 {
			return nil, fmt.Errorf("format() needs a format string")
		}
		out := Stringify(args[0])
		for i, a := range args[1:] {
			out = strings.ReplaceAll(out, fmt.Sprintf("{%d}", i), Stringify(a))
		}
		return out, nil
	case "startswith":
		if len(args) != 2 {
			return nil, fmt.Errorf("startsWith() takes 2 arguments")
		}
		return strings.HasPrefix(Stringify(args[0]), Stringify(args[1])), nil
	case "endswith":
		if len(args) != 2 {
			return nil, fmt.Errorf("endsWith() takes 2 arguments")
		}
		return strings.HasSuffix(Stringify(args[0]), Stringify(args[1])), nil
	case "contains":
		if len(args) != 2 {
			return nil, fmt.Errorf("contains() takes 2 arguments")
		}
		return strings.Contains(Stringify(args[0]), Stringify(args[1])), nil
	}
	return nil, fmt.Errorf("unsupported function %s(); this evaluator implements only the subset "+
		"the workflows here use, and fails closed on the rest", name)
}

// Truthy applies GitHub's falsy set: false, 0, the empty string, and null.
func Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	}
	return true
}

// Stringify renders a value the way GitHub interpolates it into a string.
func Stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// looseEqual is GitHub's `==`: null compares equal to the empty string, to zero
// and to false, and everything else compares by its string rendering, which is
// exact for the string-vs-string comparisons the workflows here make.
func looseEqual(a, b any) bool {
	if a == nil || b == nil {
		other := a
		if a == nil {
			other = b
		}
		return other == nil || !Truthy(other)
	}
	if ab, ok := a.(bool); ok {
		return ab == Truthy(b)
	}
	if bb, ok := b.(bool); ok {
		return bb == Truthy(a)
	}
	return Stringify(a) == Stringify(b)
}

// EvalIf evaluates a job's or a step's `if:` condition the way GitHub decides
// whether to run it. The value may be written bare or as one `${{ }}`
// expression. A condition that names no status-check function is evaluated as
// `success() && (<condition>)`, GitHub's implicit default, which is what skips
// a job whose need failed or was skipped however its own condition reads; the
// context therefore states success() whenever it is asked about such a
// condition.
func EvalIf(raw string, ctx map[string]any) (bool, error) {
	expr := strings.TrimSpace(raw)
	if strings.Contains(expr, "${{") {
		if !strings.HasPrefix(expr, "${{") || !strings.HasSuffix(expr, "}}") ||
			strings.Count(expr, "${{") != 1 {
			return false, fmt.Errorf("condition %q is not one ${{ }} expression; GitHub reads "+
				"an expression with text around it as a non-empty string, which is always true", raw)
		}
		expr = expr[3 : len(expr)-2]
	}
	toks, err := tokenizeExpr(expr)
	if err != nil {
		return false, fmt.Errorf("in `%s`: %w", strings.TrimSpace(expr), err)
	}
	if !namesStatusFunc(toks) {
		expr = "success() && (" + expr + ")"
	}
	v, err := evalExpr(expr, ctx)
	if err != nil {
		return false, err
	}
	return Truthy(v), nil
}

// namesStatusFunc reports whether a condition calls a status-check function,
// which turns GitHub's implicit success() off.
func namesStatusFunc(toks []exprToken) bool {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].kind == "ident" && statusFuncs[strings.ToLower(toks[i].text)] &&
			toks[i+1].kind == "op" && toks[i+1].text == "(" {
			return true
		}
	}
	return false
}
