package guard

import (
	"regexp"
	"strings"
)

// arithmeticEntryID: the command holds a place where bash evaluates text as
// arithmetic, with an operand the line does not fix to a literal number.
const arithmeticEntryID = "arithmetic-operand-unfixed"

// Bash evaluates the operand of an arithmetic context as an expression, and
// it evaluates a name it meets there as an expression in turn, so a variable,
// a quoted word or a subscript in one can carry a command substitution that
// runs although the line never names it: `[[ $n -gt 0 ]]` with n holding
// `a[$(cmd)]` runs cmd. The approver therefore admits an arithmetic context
// only when everything in it is a literal number and operators, and refuses
// every other one. The scan reads the command text whole, quotes and
// here-document bodies included: a context inside quotes is read as one, so
// nothing a quote or a here-document hides can carry one past it, and the cost
// is a refusal of text that only quotes such a context, which the agent can
// write another way. `[` and `test` compare numbers without evaluating them,
// so they stay open for a comparison on a value the line does not fix.

// arithmeticLiteral is every byte an arithmetic operand the approver admits
// may hold: digits, white space and the operators, and no name, quote,
// expansion or subscript. The `;` separates a for-loop header's parts.
const arithmeticLiteral = "0123456789 \t\n+-*/%<>=!&|^~?:,;()"

// arithmeticSubscriptLiteral is arithmeticLiteral for a subscript, which may
// also be `@` or `*` for the whole array.
const arithmeticSubscriptLiteral = arithmeticLiteral + "@"

var (
	// arithmeticCompare is an arithmetic comparison inside `[[ … ]]`, and
	// `-v`, whose operand may be a subscripted name.
	arithmeticCompare = regexp.MustCompile(`(^|\s)-(eq|ne|lt|le|gt|ge|v)(\s|$)`)
	// arithmeticCommand is `let`, or a declaration naming the integer or the
	// name-reference attribute (`declare -i`, `local -ai`, `typeset -n`),
	// standing where a command may start.
	arithmeticCommand = regexp.MustCompile(`(^|[\s;&|(){}` + "`" + `])(let(\s|$)|(declare|typeset|local|readonly|export)(\s+[^;&|\n]*)?\s-[A-Za-z]*[in][A-Za-z]*(\s|$))`)
	// declaration is a builtin that takes variable names, standing where a
	// command may start, and the rest of its command.
	declaration = regexp.MustCompile(`(^|[\s;&|(){}` + "`" + `])(declare|typeset|local|readonly|export|read|unset|mapfile|readarray|getopts)(\s[^;&|\n]*)?`)
	// printfTarget is `printf -v` or `wait -p` and the name it assigns.
	printfTarget = regexp.MustCompile(`(^|[\s;&|(){}` + "`" + `])(printf\s+(-[^v\s]\S*\s+)*-v|wait\s+(-[^p\s]\S*\s+)*-p)\s*(\S+)`)
	// testNamed is `[ -v name ]` or `test -v name`, which reads a name.
	testNamed = regexp.MustCompile(`(^|[\s;&|(){}` + "`" + `])(\[|test)\s[^;&|\n]*-v\s+(\S+)`)
	// identifier is a name a builtin may be handed without an index.
	identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	// arithmeticSubscriptOpen is a name followed by a subscript, in an
	// assignment (`a[i]=`), an expansion (`${a[i]}`) or a name a builtin is
	// handed (`printf -v 'a[i]'`, `unset 'a[i]'`).
	arithmeticSubscriptOpen = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*\[`)
)

// arithmeticUnfixed reports why command holds an arithmetic context whose
// operand is not a literal number, or "" when it holds none.
func arithmeticUnfixed(command string) string {
	if why := unfixedArithmeticBodies(command); why != "" {
		return why
	}
	if why := unfixedDoubleBrackets(command); why != "" {
		return why
	}
	if arithmeticCommand.MatchString(command) {
		return "it runs `let` or declares an integer variable, and bash evaluates every value such a variable is given as arithmetic"
	}
	if why := unfixedNames(command); why != "" {
		return why
	}
	if why := unfixedSubscripts(command); why != "" {
		return why
	}
	return unfixedSubstrings(command)
}

// unfixedNames reports a builtin handed a variable name the line does not
// fix: bash reads a name it is handed as a possibly indexed name, so `read
// "$n"`, `unset "$n"`, `local "$n=1"`, `printf -v "$n"` and `[ -v "$n" ]`
// evaluate an index n holds as arithmetic. A name is fixed when it is a plain
// identifier; a declaration's value after its `=` is not a name and is left
// alone.
func unfixedNames(command string) string {
	const why = "it hands a builtin a variable name that is not written in the line, and bash evaluates an index in such a name as arithmetic"
	for _, m := range printfTarget.FindAllStringSubmatch(command, -1) {
		if !identifier.MatchString(unquoteWord(m[5])) {
			return why
		}
	}
	for _, m := range testNamed.FindAllStringSubmatch(command, -1) {
		if !identifier.MatchString(unquoteWord(m[3])) {
			return why
		}
	}
	for _, m := range declaration.FindAllStringSubmatch(command, -1) {
		words := shellWords(m[3])
		for k := 0; k < len(words); k++ {
			w := words[k]
			// A redirection's target is a path, not a name: `mapfile -t a
			// < "$f"`, `read -r x <<< "$y"`.
			if op := strings.TrimLeft(w, "0123456789&"); op != "" && (op[0] == '<' || op[0] == '>') {
				if strings.Trim(op, "<>&|-") == "" {
					k++
				}
				continue
			}
			name := w
			if eq := strings.IndexByte(w, '='); eq >= 0 {
				name = w[:eq]
			}
			if strings.ContainsAny(name, "$`") {
				return why
			}
		}
	}
	return ""
}

// unquoteWord is w without one pair of quotes around the whole of it.
func unquoteWord(w string) string {
	if len(w) >= 2 && (w[0] == '\'' || w[0] == '"') && w[len(w)-1] == w[0] {
		return w[1 : len(w)-1]
	}
	return w
}

// shellWords splits s at unquoted white space, keeping single- and
// double-quoted runs inside the word they stand in.
func shellWords(s string) []string {
	var words []string
	var cur strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			cur.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			cur.WriteByte(c)
		case c == ' ' || c == '\t' || c == '\n':
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

// unfixedArithmeticBodies reads every `((` (the command, a for-loop header
// and the `$((` expansion) and every `$[`, to its matching close, and reports
// one whose body is not literal; a body with no close is reported too.
func unfixedArithmeticBodies(command string) string {
	for i := 0; i+1 < len(command); i++ {
		switch {
		case command[i] == '(' && command[i+1] == '(':
			body, ok := balancedBody(command, i+2, '(', ')', 2)
			if !ok || !onlyBytes(body, arithmeticLiteral) {
				return "it holds an arithmetic expression `((…))` with a name, a quote or an expansion in it, which bash evaluates as arithmetic"
			}
			i += 1 + len(body)
		case command[i] == '$' && command[i+1] == '[':
			body, ok := balancedBody(command, i+2, '[', ']', 1)
			if !ok || !onlyBytes(body, arithmeticLiteral) {
				return "it holds an arithmetic expression `$[…]` with a name, a quote or an expansion in it, which bash evaluates as arithmetic"
			}
			i += 1 + len(body)
		}
	}
	return ""
}

// unfixedDoubleBrackets reports a `[[ … ]]` holding an arithmetic comparison
// or `-v`, whose operands bash evaluates as arithmetic, unless every operand
// is literal.
func unfixedDoubleBrackets(command string) string {
	for at := 0; ; {
		i := strings.Index(command[at:], "[[")
		if i < 0 {
			return ""
		}
		start := at + i + 2
		end := strings.Index(command[start:], "]]")
		if end < 0 {
			end = len(command) - start
		}
		body := command[start : start+end]
		if arithmeticCompare.MatchString(body) && !onlyBytes(arithmeticCompare.ReplaceAllString(body, " "), arithmeticLiteral) {
			return "it compares numbers inside `[[ … ]]`, whose operands bash evaluates as arithmetic; `[ … ]` compares them without evaluating them"
		}
		at = start + end
		if at >= len(command) {
			return ""
		}
	}
}

// unfixedSubscripts reports a subscript whose index is not literal: on a
// name (`a[i]`) and in a compound assignment (`a=([i]=x)`), both of which
// bash evaluates as arithmetic for an indexed array.
func unfixedSubscripts(command string) string {
	for _, loc := range arithmeticSubscriptOpen.FindAllStringIndex(command, -1) {
		if !literalSubscript(command, loc[1]) {
			return "it indexes an array with a name, a quote or an expansion, and bash evaluates an index as arithmetic"
		}
	}
	for at := 0; ; {
		i := strings.Index(command[at:], "=(")
		if i < 0 {
			return ""
		}
		start := at + i + 2
		body, ok := balancedBody(command, start, '(', ')', 1)
		if !ok {
			body = command[start:]
		}
		for j := 0; j < len(body); j++ {
			if body[j] == '[' && !literalSubscript(body, j+1) {
				return "it assigns an array with an index that is a name, a quote or an expansion, and bash evaluates an index as arithmetic"
			}
		}
		at = start + len(body)
		if at >= len(command) {
			return ""
		}
	}
}

// literalSubscript reports whether the subscript opening just before
// text[from] closes, and holds only a literal index.
func literalSubscript(text string, from int) bool {
	end := strings.IndexByte(text[from:], ']')
	return end >= 0 && onlyBytes(text[from:from+end], arithmeticSubscriptLiteral)
}

// unfixedSubstrings reports a substring expansion `${v:offset:length}` whose
// offset or length is not literal: bash evaluates both as arithmetic. The
// operators `:-`, `:=`, `:+` and `:?` are not substrings and are left alone.
func unfixedSubstrings(command string) string {
	for at := 0; ; {
		i := strings.Index(command[at:], "${")
		if i < 0 {
			return ""
		}
		start := at + i + 2
		body, ok := balancedBody(command, start, '{', '}', 1)
		if !ok {
			body = command[start:]
		}
		if strings.HasPrefix(body, "!") {
			return "it expands a variable named by another variable `${!v}`, and bash evaluates an index in that name as arithmetic"
		}
		name := 0
		for name < len(body) && (body[name] == '_' || body[name] == '#' ||
			body[name] >= '0' && body[name] <= '9' || body[name] >= 'A' && body[name] <= 'Z' || body[name] >= 'a' && body[name] <= 'z') {
			name++
		}
		if name < len(body) && body[name] == ':' && (name+1 >= len(body) || !strings.ContainsRune("-=+?", rune(body[name+1]))) {
			if !onlyBytes(body[name+1:], arithmeticLiteral) {
				return "it takes a substring `${v:offset:length}` at an offset or a length that is not a number, and bash evaluates both as arithmetic"
			}
		}
		at = start + len(body)
		if at >= len(command) {
			return ""
		}
	}
}

// balancedBody returns the text from text[from] up to the byte before the
// close that brings depth, counted from the opening already read, to zero,
// and whether that close was found. Every open and close byte counts,
// whatever quotes stand around it, the way the scan reads the text whole.
func balancedBody(text string, from int, open, close byte, depth int) (string, bool) {
	for j := from; j < len(text); j++ {
		switch text[j] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				// A `((` closes on `))`: the body stops before the first of
				// the pair.
				if open == '(' && j > from && text[j-1] == ')' {
					return text[from : j-1], true
				}
				return text[from:j], true
			}
		}
	}
	return text[from:], false
}

// onlyBytes reports whether every byte of s is one of allowed.
func onlyBytes(s, allowed string) bool {
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(allowed, s[i]) < 0 {
			return false
		}
	}
	return true
}
